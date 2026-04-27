"""AI-powered question generation service."""

import json
import logging
from typing import Optional

from src.core.llm_client import LLMClient
from src.core.prompt_templates import PromptTemplateService
from src.models.responses import QuestionGenerationResponse, QuestionItem
from src.utils.validators import validate_difficulty, validate_question_count, validate_topic

logger = logging.getLogger(__name__)

DEFAULT_QUESTION_TYPES = [
    "multiple_choice",
    "open_ended",
    "true_false",
    "short_answer",
]


class QuestionService:
    """Service for generating assessment questions using LLM."""

    def __init__(
        self,
        llm_client: LLMClient,
        prompt_templates: PromptTemplateService,
    ) -> None:
        self._llm = llm_client
        self._templates = prompt_templates

    async def generate_questions(
        self,
        topic: str,
        context: Optional[str] = None,
        num_questions: int = 5,
        difficulty: str = "medium",
        question_types: Optional[list[str]] = None,
    ) -> QuestionGenerationResponse:
        """Generate assessment questions for the given topic.

        Args:
            topic: The subject topic for questions.
            context: Optional additional context.
            num_questions: Number of questions to generate.
            difficulty: Difficulty level (easy, medium, hard).
            question_types: Types of questions to generate.

        Returns:
            QuestionGenerationResponse with generated questions.

        Raises:
            ValueError: If input parameters are invalid.
        """
        # Validate inputs
        topic = validate_topic(topic)
        difficulty = validate_difficulty(difficulty)
        num_questions = validate_question_count(num_questions)
        context = context.strip() if context else "No additional context provided."

        if question_types is None:
            question_types = DEFAULT_QUESTION_TYPES

        types_str = ", ".join(question_types)

        # Build prompt
        template = self._templates.get_template("question_generation")
        prompt, system_prompt = template.format(
            num_questions=str(num_questions),
            topic=topic,
            context=context,
            difficulty=difficulty,
        )

        # Enhance prompt with question type constraints
        prompt = prompt.replace(
            '"type": "multiple_choice|open_ended|true_false|short_answer"',
            f'"type": "{types_str}"',
        )

        logger.info(
            "Generating %d questions for topic='%s' difficulty='%s'",
            num_questions,
            topic,
            difficulty,
        )

        # Call LLM
        raw_response = await self._llm.generate_json(
            prompt=prompt,
            system_prompt=system_prompt,
        )

        # Parse and validate response
        questions = self._parse_questions(raw_response, difficulty)

        return QuestionGenerationResponse(
            questions=questions,
            topic=topic,
            total_count=len(questions),
        )

    def _parse_questions(
        self, raw: dict, default_difficulty: str
    ) -> list[QuestionItem]:
        """Parse LLM response into QuestionItem objects.

        Args:
            raw: Raw JSON response from LLM.
            default_difficulty: Fallback difficulty level.

        Returns:
            List of parsed QuestionItem objects.
        """
        questions_data = raw.get("questions", [])
        if not questions_data:
            # Try alternative key structures
            if isinstance(raw.get("data"), list):
                questions_data = raw["data"]
            elif isinstance(raw.get("items"), list):
                questions_data = raw["items"]

        questions = []
        for item in questions_data:
            question = QuestionItem(
                text=item.get("text", ""),
                type=item.get("type", "open_ended"),
                difficulty=item.get("difficulty", default_difficulty),
                expected_answer=item.get("expected_answer", ""),
                options=item.get("options"),
                explanation=item.get("explanation"),
            )
            questions.append(question)

        if not questions:
            logger.warning("LLM returned no parseable questions")

        return questions
