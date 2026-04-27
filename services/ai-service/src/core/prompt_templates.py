"""Prompt templates for various AI operations."""

from dataclasses import dataclass
from string import Template
from typing import Optional


@dataclass(frozen=True)
class PromptTemplate:
    """Immutable prompt template with placeholder substitution."""

    name: str
    template: str
    system_prompt: Optional[str] = None

    def format(self, **kwargs: str) -> tuple[str, Optional[str]]:
        """Format the template with the given variables.

        Returns:
            Tuple of (formatted user prompt, system prompt).
        """
        tmpl = Template(self.template)
        return tmpl.safe_substitute(**kwargs), self.system_prompt


# Question generation templates
QUESTION_GENERION_TEMPLATE = PromptTemplate(
    name="question_generation",
    template="""Сгенерируй $num_questions вопросов по теме "$topic" со сложностью: $difficulty.

Контекст:
$context

Отвечай только на русском языке.
Верни результат как JSON-объект со следующей структурой:
{
    "questions": [
        {
            "text": "Текст вопроса",
            "type": "multiple_choice|open_ended|true_false|short_answer",
            "difficulty": "easy|medium|hard",
            "expected_answer": "Ожидаемый ответ или правильный вариант",
            "options": ["вариант A", "вариант B", "вариант C", "вариант D"],
            "explanation": "Краткое объяснение правильного ответа"
        }
    ]
}

Для multiple_choice укажи 4 варианта и один правильный.
Для true_false укажи правильный ответ как "true" или "false".
Для open_ended и short_answer укажи образцовый ответ.""",
    system_prompt="Ты экспертный преподаватель. Всегда отвечай только на русском языке.",
)

ANALYSIS_TEMPLATE = PromptTemplate(
    name="answer_analysis",
    template="""Проанализируй ответ студента и дай подробную оценку.

Вопрос: $question
Ответ студента: $answer
Ожидаемый ответ: $expected_answer

$rubric_context

Отвечай только на русском языке.

Оцени ответ по критериям:
1. Correctness (0-100): Насколько ответ фактически корректен?
2. Completeness (0-100): Насколько полно покрыт ожидаемый ответ?
3. Clarity (0-100): Насколько ответ понятный и структурированный?
4. Relevance (0-100): Насколько ответ релевантен вопросу?

Правила оценки:
- 90-100 ставь только если ответ почти полностью совпадает с ожидаемым смыслом и без существенных ошибок.
- Если ответ уклончивый, общий, содержит "не знаю"/"кажется"/"примерно" или уходит в сторону, correctness и relevance должны быть заметно ниже.
- Если ответ не раскрывает ключевые пункты ожидаемого ответа, completeness не может быть высокой.
- Не завышай score из-за хорошего стиля, если по сути ответ слабый.
- Используй всю шкалу 0-100; не округляй слабые ответы к 70-100.

Верни JSON-объект со структурой:
{
    "scores": {
        "correctness": 85,
        "completeness": 70,
        "clarity": 90,
        "relevance": 95
    },
    "overall_score": 85,
    "feedback": "Подробная обратная связь для студента",
    "strengths": ["Сильная сторона 1", "Сильная сторона 2"],
    "weaknesses": ["Зона роста 1"],
    "suggested_improvements": ["Рекомендация 1"]
}""",
    system_prompt="Ты экспертный проверяющий. Всегда отвечай только на русском языке.",
)

INTERVIEWER_TURN_TEMPLATE = PromptTemplate(
    name="interviewer_turn",
    template="""Контекст:
- Роль: $role
- Вакансия: $vacancy_title
- Категория: $vacancy_category
- Режим интервью: $interview_mode
- Подсказка по режиму: $mode_guidance
- Стиль интервью: $style_profile
- Сжатый контекст сессии:
$session_context
- Последние темы:
$recent_topics
- Практический фокус: $practice_focus
- Теоретический фокус: $theory_focus
- Ключевые навыки: $primary_skills
- Области внимания: $focus_areas
- Уровень: $level
- Текущая тема: $topic
- Сложность: $difficulty
- Давление: $pressure
- Ограничения: осталось $time_left_sec сек, $questions_left вопросов
- Идентификатор хода: $turn_nonce
- История беседы:
$history
- Оценка последнего ответа:
$answer_guidance
- Режим ответа на последний ход:
$response_policy

- Нельзя повторять формулировки из списка:
$avoid_questions
- Повторы запрещены не только дословно, но и по смыслу; если вопрос слишком близок к списку выше, выбери другой угол, другую тему или другую глубину.

Последний ответ кандидата:
$candidate_answer

Задача:
1) Внутренне проанализируй ответ кандидата и предыдущие ходы
2) Определи, нужно ли:
   - задать follow-up,
   - уточнить слабое место,
   - сменить тему,
   - усилить давление
3) Сгенерируй ровно один следующий ход:
    - в режиме theory: следующий теоретический вопрос/уточнение
    - в режиме practice: проверка решения и следующее coding-задание
4) Не повторяй уже заданные вопросы

Верни ТОЛЬКО JSON по согласованной схеме.
""",
    system_prompt="""Ты технический интервьюер и симулятор живого собеседования.

Твоя задача: вести реалистичное техническое интервью.

Обязательные правила:
- Всегда отвечай только на русском языке.
- Задавай ровно один следующий вопрос, уточнение или смену темы.
- Не начинай ответ с похвалы, шаблонных вступлений или общих фраз вроде "хорошо", "отлично", "понятно".
- Учитывай роль, уровень, текущую тему, давление и историю беседы.
- Используй "Сжатый контекст сессии" как главный источник фактов о сессии; историю используй для уточнения и анти-повтора.
- Если последние темы указывают на зацикливание, обязательно переключись на соседнюю область, другую глубину или другой тип проверки.
- Сохраняй контекст предыдущих ответов и ссылайся на них, когда это уместно.
- Меняй тему динамически, если тема исчерпана, ответ слишком слабый или требуется проверить смежную область.
- Повышай давление постепенно: от простого уточнения к trade-offs, edge cases, scale, reliability и production concerns.
- Не повторяй формулировки из блока "Нельзя повторять формулировки из списка".
- Если генерируемый вопрос слишком похож на любой пункт из блока "Нельзя повторять формулировки из списка", немедленно перестрой вопрос так, чтобы он проверял другое знание.
- Если режим practice, веди live-coding: давай задание, проверяй решение кандидата, при верном решении переходи к следующему заданию, при неверном дай правильный ответ и предложи улучшение.
- Если режим theory, задавай только вопросы по техтеории в выбранном направлении и уровне.

Как вести диалог:
- Если ответ кандидата слабый, попроси уточнить reasoning, trade-offs, ограничения или проверку результата.
- Если последний ответ слабый или уклончивый, обязательно задавай follow-up, а не переходи к новой теме.
- Для слабого ответа формулируй один короткий прямой follow-up и удерживай фокус на одном пробеле, а не на всей теме сразу.
- Не повышай сложность или оценку без опоры на факты из ответа кандидата: если ответ слабый или уклончивый, задай уточняющий вопрос вместо похвалы.
- Если кандидат ответил мимо темы, не делай вид, что ответ был корректным: сразу верни к сути.
- Если кандидат просит помочь или объяснить, задай более точный follow-up и удерживай фокус интервью.
- Если кандидат уходит в сторону, верни его к теме или мягко переключи на соседнюю область.
- Если нужно, ссылайся на предыдущий ответ кандидата, чтобы диалог выглядел живым.

Тон:
- коротко
- профессионально
- уверенно
- по-деловому
- как на реальном интервью

Формат ответа:
- верни только JSON по согласованной схеме
- поле question должно содержать один естественный вопрос или уточнение
- поле topic должно отражать реальную тему следующего хода
- difficulty_delta и pressure_level должны соответствовать ходу интервью
- flags должны честно отражать характер ответа
""",
)

SIMILARITY_SYSTEM = "You compare the semantic meaning of two texts."


class PromptTemplateService:
    """Service for managing and retrieving prompt templates."""

    def __init__(self) -> None:
        self._templates: dict[str, PromptTemplate] = {
            "question_generation": QUESTION_GENERION_TEMPLATE,
            "answer_analysis": ANALYSIS_TEMPLATE,
            "interviewer_turn": INTERVIEWER_TURN_TEMPLATE,
        }

    def get_template(self, name: str) -> PromptTemplate:
        """Get a prompt template by name.

        Args:
            name: Template name.

        Returns:
            The PromptTemplate instance.

        Raises:
            KeyError: If template name not found.
        """
        if name not in self._templates:
            raise KeyError(f"Unknown template: {name}")
        return self._templates[name]

    def register_template(self, template: PromptTemplate) -> None:
        """Register a new prompt template.

        Args:
            template: The PromptTemplate to register.
        """
        self._templates[template.name] = template

    def list_templates(self) -> list[str]:
        """Return list of registered template names."""
        return list(self._templates.keys())
