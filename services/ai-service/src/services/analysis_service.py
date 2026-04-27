"""Answer analysis service for evaluating student responses."""

import logging
import re
from typing import Optional

from src.core.embeddings import EmbeddingService
from src.core.llm_client import LLMClient
from src.core.prompt_templates import PromptTemplateService
from src.models.responses import AnalysisResponse, AnalysisScores
from src.utils.helpers import safe_get

logger = logging.getLogger(__name__)
LATIN_RE = re.compile(r"[A-Za-z]")
DEFLECTION_RE = re.compile(
    r"\b(не\s+знаю|не\s+уверен|не\s+помню|затрудняюсь|кажется|примерно|скорее\s+всего|не\s+могу\s+сказать)\b",
    re.IGNORECASE,
)


class AnalysisService:
    """Service for analyzing student answers using LLM and embeddings."""

    def __init__(
        self,
        llm_client: LLMClient,
        prompt_templates: PromptTemplateService,
        embedding_service: EmbeddingService,
    ) -> None:
        self._llm = llm_client
        self._templates = prompt_templates
        self._embeddings = embedding_service

    async def analyze_answer(
        self,
        question: str,
        answer: str,
        expected_answer: str,
        rubric: Optional[str] = None,
    ) -> AnalysisResponse:
        """Analyze a student's answer against an expected answer.

        Args:
            question: The original question.
            answer: The student's answer.
            expected_answer: The model/correct answer.
            rubric: Optional custom grading rubric.

        Returns:
            AnalysisResponse with scores and feedback.
        """
        rubric_context = (
            f"Дополнительная рубрика оценивания: {rubric}"
            if rubric
            else "Используй стандартные академические критерии оценивания."
        )

        template = self._templates.get_template("answer_analysis")
        prompt, system_prompt = template.format(
            question=question,
            answer=answer,
            expected_answer=expected_answer,
            rubric_context=rubric_context,
        )

        logger.info("Analyzing answer for question: %s...", question[:50])

        raw = await self._llm.generate_json(
            prompt=prompt,
            system_prompt=system_prompt,
        )

        analysis = self._parse_analysis(raw)
        return await self._apply_guardrails(
            analysis=analysis,
            question=question,
            answer=answer,
            expected_answer=expected_answer,
        )

    async def compute_similarity(
        self, text_a: str, text_b: str
    ) -> float:
        """Compute semantic similarity between two texts.

        Args:
            text_a: First text.
            text_b: Second text.

        Returns:
            Similarity score between 0.0 and 1.0.
        """
        return await self._embeddings.similarity(text_a, text_b)

    def _parse_analysis(self, raw: dict) -> AnalysisResponse:
        """Parse LLM analysis response into a structured model.

        Args:
            raw: Raw JSON from LLM.

        Returns:
            Structured AnalysisResponse.
        """
        scores_data = raw.get("scores", {})

        scores = AnalysisScores(
            correctness=safe_get(scores_data, "correctness", default=50),
            completeness=safe_get(scores_data, "completeness", default=50),
            clarity=safe_get(scores_data, "clarity", default=50),
            relevance=safe_get(scores_data, "relevance", default=50),
        )

        feedback = raw.get("feedback", "Обратная связь недоступна.")
        strengths = raw.get("strengths", [])
        weaknesses = raw.get("weaknesses", [])
        suggested_improvements = raw.get("suggested_improvements", [])

        if self._contains_latin(feedback):
            feedback = "Ответ оценен. Пожалуйста, уточните детали и добавьте больше контекста по решению."

        if any(self._contains_latin(item) for item in strengths):
            strengths = ["Ответ соответствует теме вопроса."]

        if any(self._contains_latin(item) for item in weaknesses):
            weaknesses = ["Ответу может не хватать полноты или конкретики."]

        if any(self._contains_latin(item) for item in suggested_improvements):
            suggested_improvements = [
                "Добавьте больше деталей и обоснование выбранного решения.",
                "Укажите компромиссы и возможные ограничения подхода.",
            ]

        return AnalysisResponse(
            scores=scores,
            overall_score=raw.get("overall_score", 50),
            feedback=feedback,
            strengths=strengths,
            weaknesses=weaknesses,
            suggested_improvements=suggested_improvements,
        )

    async def _apply_guardrails(
        self,
        analysis: AnalysisResponse,
        question: str,
        answer: str,
        expected_answer: str,
    ) -> AnalysisResponse:
        answer_text = (answer or "").strip()
        question_text = (question or "").strip()
        expected_text = (expected_answer or "").strip()

        answer_words = self._tokenize(answer_text)
        answer_word_count = len(answer_words)
        deflective = bool(DEFLECTION_RE.search(answer_text))

        try:
            similarity_to_question = await self.compute_similarity(answer_text, question_text)
        except Exception:
            similarity_to_question = self._lexical_overlap(question_text, answer_text)

        try:
            similarity_to_expected = await self.compute_similarity(answer_text, expected_text)
        except Exception:
            similarity_to_expected = self._lexical_overlap(expected_text, answer_text)

        structure_score = self._structure_score(answer_text)
        vocabulary_ratio = self._unique_ratio(answer_words)

        correctness_cap = 20 + similarity_to_expected * 75
        relevance_cap = 20 + similarity_to_question * 70
        completeness_cap = 15 + min(55.0, answer_word_count * 3.5) + similarity_to_expected * 15
        clarity_cap = 20 + structure_score * 40 + vocabulary_ratio * 20

        if deflective:
            correctness_cap -= 20
            relevance_cap -= 25
            completeness_cap -= 20
            clarity_cap -= 10

        if answer_word_count < 8:
            correctness_cap = min(correctness_cap, 35)
            relevance_cap = min(relevance_cap, 40)
            completeness_cap = min(completeness_cap, 30)

        if similarity_to_question < 0.25:
            relevance_cap = min(relevance_cap, 40)
            correctness_cap = min(correctness_cap, 45)

        final_correctness = self._bounded_score(analysis.scores.correctness, correctness_cap)
        final_completeness = self._bounded_score(analysis.scores.completeness, completeness_cap)
        final_clarity = self._bounded_score(analysis.scores.clarity, clarity_cap)
        final_relevance = self._bounded_score(analysis.scores.relevance, relevance_cap)

        overall_cap = (correctness_cap + completeness_cap + clarity_cap + relevance_cap) / 4
        overall_score = min(analysis.overall_score, overall_cap)

        strengths = list(analysis.strengths)
        weaknesses = list(analysis.weaknesses)
        suggested_improvements = list(analysis.suggested_improvements)

        if final_relevance < 50:
            weaknesses.append("Ответ слабо связан с вопросом")
            suggested_improvements.append("Сначала прямо ответьте на вопрос, затем добавьте детали и пример")
        if final_correctness < 50:
            weaknesses.append("Не хватает точности или есть признаки ухода от сути вопроса")
            suggested_improvements.append("Проверьте факты и объясните решение через конкретные шаги")
        if final_completeness < 50:
            weaknesses.append("Ответ недостаточно раскрывает тему")
            suggested_improvements.append("Добавьте альтернативы, ограничения и trade-offs")
        if final_clarity < 50:
            weaknesses.append("Формулировка ответа слишком размытая или несистемная")
            suggested_improvements.append("Структурируйте ответ: тезис, аргументация, вывод")

        if final_relevance >= 70 and final_correctness >= 65:
            strengths.append("Ответ удерживает фокус на вопросе")
        if final_completeness >= 70:
            strengths.append("Ответ достаточно полно раскрывает тему")

        return AnalysisResponse(
            scores=AnalysisScores(
                correctness=round(final_correctness, 2),
                completeness=round(final_completeness, 2),
                clarity=round(final_clarity, 2),
                relevance=round(final_relevance, 2),
            ),
            overall_score=round(overall_score, 2),
            feedback=analysis.feedback,
            strengths=self._dedupe_texts(strengths),
            weaknesses=self._dedupe_texts(weaknesses),
            suggested_improvements=self._dedupe_texts(suggested_improvements),
        )

    @staticmethod
    def _tokenize(text: str) -> list[str]:
        return re.findall(r"[A-Za-zА-Яа-я0-9_]+", text.lower())

    @staticmethod
    def _unique_ratio(tokens: list[str]) -> float:
        if not tokens:
            return 0.0
        return len(set(tokens)) / len(tokens)

    @staticmethod
    def _structure_score(text: str) -> float:
        if not text:
            return 0.0
        paragraphs = text.count("\n")
        bullets = text.count("-") + text.count("*")
        punctuation = sum(text.count(char) for char in [".", ",", ";", ":"])
        raw = 0.25 * paragraphs + 0.25 * bullets + min(1.0, punctuation / max(1, len(text) / 40))
        return min(1.0, raw)

    @staticmethod
    def _lexical_overlap(text_a: str, text_b: str) -> float:
        tokens_a = set(AnalysisService._tokenize(text_a))
        tokens_b = set(AnalysisService._tokenize(text_b))
        if not tokens_a or not tokens_b:
            return 0.0
        shared = len(tokens_a & tokens_b)
        total = len(tokens_a | tokens_b)
        return shared / total if total else 0.0

    @staticmethod
    def _bounded_score(value: float, cap: float) -> float:
        return max(0.0, min(float(value), max(0.0, cap)))

    @staticmethod
    def _dedupe_texts(values: list[str]) -> list[str]:
        result: list[str] = []
        seen: set[str] = set()
        for item in values:
            normalized = item.strip()
            if not normalized or normalized in seen:
                continue
            seen.add(normalized)
            result.append(normalized)
        return result

    @staticmethod
    def _contains_latin(value: str) -> bool:
        return bool(LATIN_RE.search(value or ""))
