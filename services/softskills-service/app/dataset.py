"""Dataset utilities.

Loads the seed dataset (~644 samples) and augments it deterministically:

  - Low-quality answer templates (target 0.05–0.15)
  - Mid-quality answer templates (target 0.40–0.55)
  - High-quality answer templates (target 0.85–0.95)

Applied to each unique question. This roughly triples the seed and
balances the target distribution so the regressor doesn't collapse
to "everyone is mediocre" — a real risk on the original ~644-sample
seed where most targets cluster around 0.4–0.7.
"""

from __future__ import annotations

import json
import random
from pathlib import Path
from typing import List, TypedDict


class Sample(TypedDict):
    question: str
    answer: str
    target: float


# Hard-coded answer fragments grouped by quality tier. They're written
# to be plausible across topics — soft-skill interviewers grade
# *structure* (specific example + measurable outcome + reflection) more
# than *content*, so the tier signal generalises well.
LOW_TEMPLATES = [
    "Не знаю, как-то само получается.",
    "Обычно никак не реагирую, жду пока всё решится само.",
    "Стараюсь не думать об этом.",
    "Это сложный вопрос, у меня нет ответа.",
    "Чаще всего просто делаю как сказали.",
    "Не сталкивался с таким, поэтому ничего не скажу.",
    "Это зависит от ситуации, конкретно не могу описать.",
    "Просто работаю как все, ничего особенного.",
    "Если честно, не задумывался об этом раньше.",
    "Стараюсь избегать таких ситуаций.",
]

MID_TEMPLATES = [
    "Я стараюсь сначала понять контекст, а потом принимаю решение по обстановке.",
    "Обычно обсуждаю с коллегами и согласовываем подход.",
    "Я предпочитаю спокойно проговорить с командой и найти компромисс.",
    "В первую очередь я разбираюсь в причинах, а потом действую.",
    "Стараюсь делать всё аккуратно и проверять результат.",
    "Я слушаю мнение других и стараюсь найти общее решение.",
    "Делю задачу на части и иду по приоритетам.",
    "Беру время на анализ, а потом принимаю решение.",
    "Если что-то идёт не так, я открыто говорю об этом руководителю.",
    "Я ставлю реалистичные цели и регулярно проверяю прогресс.",
]

HIGH_TEMPLATES = [
    "На прошлом проекте: команда из 6 человек, дедлайн через 3 недели, я ввёл daily-стендапы по 15 минут, "
    "выкатили в срок и сократили блокеры на 40%.",
    "Когда у нас был конфликт между фронт и бэк по контракту API, "
    "я организовал встречу с конкретными use-cases, мы согласовали схему за 1 час "
    "и потом договорились фиксировать решения в ADR.",
    "В кризисной ситуации с упавшим прод-сервисом я взял координацию: "
    "распределил роли (incident lead, communicator, fixer), записывали действия в Slack-канал, "
    "восстановили SLA за 18 минут вместо обычных 40.",
    "Я разбиваю большую задачу на 3-4 шага по 1-2 дня, оцениваю риски, "
    "пишу план в Notion и регулярно даю апдейт стейкхолдерам — это снижает неопределённость и тревогу команды.",
    "Когда я перешёл в новую команду, я первый месяц только слушал, "
    "брал 1:1 с каждым, делал заметки, и только потом предложил три конкретных улучшения процесса с метриками.",
    "Я применяю STAR (Situation, Task, Action, Result): описал ситуацию, "
    "обозначил задачу, объяснил действия и привёл измеримый результат — это помогает удерживать фокус разговора.",
    "Для мотивации команды я провёл ретроспективу и выяснил, что людям не хватает признания — "
    "ввёл еженедельные shout-outs в Slack и связку с премиями. Текучка снизилась на 25% за квартал.",
    "Когда заказчик постоянно менял требования, я предложил перейти на двухнедельные итерации с фиксированным scope. "
    "За полгода поставили в срок 11 из 12 спринтов.",
    "Я разрешил конфликт двух senior-разработчиков, предложив pair-программирование по 2 часа в день. "
    "Они нашли общий язык и через месяц сами стали менторить друг друга.",
    "Делая ревью кода, я всегда даю конкретные ссылки на стандарты и предлагаю альтернативу, "
    "а не просто 'это неправильно'. Это снизило время на доработку PR в среднем на 30%.",
]


def load_seed(seed_path: Path) -> List[Sample]:
    with open(seed_path, "r", encoding="utf-8") as f:
        raw = json.load(f)
    if isinstance(raw, dict) and "questions" in raw:
        raw = raw["questions"]
    out: List[Sample] = []
    for item in raw:
        if not isinstance(item, dict):
            continue
        q = str(item.get("question", "")).strip()
        a = str(item.get("answer", "")).strip()
        t = item.get("target")
        if not q or not a or t is None:
            continue
        try:
            tv = float(t)
        except (TypeError, ValueError):
            continue
        if tv < 0:
            tv = 0.0
        if tv > 1:
            tv = 1.0
        out.append({"question": q, "answer": a, "target": tv})
    return out


def load_question_pool(pool_path: Path) -> List[str]:
    with open(pool_path, "r", encoding="utf-8") as f:
        raw = json.load(f)
    items = raw.get("questions", raw) if isinstance(raw, dict) else raw
    qs: List[str] = []
    seen = set()
    for it in items:
        q = it.get("question") if isinstance(it, dict) else str(it)
        if not q:
            continue
        q = str(q).strip()
        if not q or q in seen:
            continue
        seen.add(q)
        qs.append(q)
    return qs


def augment(samples: List[Sample], questions: List[str], rng: random.Random) -> List[Sample]:
    augmented = list(samples)

    def jitter(lo: float, hi: float) -> float:
        return round(lo + rng.random() * (hi - lo), 3)

    # 1. For every unique question in the seed, also add one synthetic
    #    sample per quality tier — keeps the regressor exposed to
    #    canonical "bad / mid / great" patterns at all topics.
    seen_q = {s["question"] for s in samples}
    for q in seen_q:
        augmented.append({"question": q, "answer": rng.choice(LOW_TEMPLATES),  "target": jitter(0.05, 0.18)})
        augmented.append({"question": q, "answer": rng.choice(MID_TEMPLATES),  "target": jitter(0.42, 0.58)})
        augmented.append({"question": q, "answer": rng.choice(HIGH_TEMPLATES), "target": jitter(0.85, 0.96)})

    # 2. Cover questions from the pool that aren't in the seed — give
    #    each one tier examples so the model handles them out of the box.
    extra_qs = [q for q in questions if q not in seen_q]
    rng.shuffle(extra_qs)
    for q in extra_qs[:200]:  # keep growth bounded
        augmented.append({"question": q, "answer": rng.choice(LOW_TEMPLATES),  "target": jitter(0.05, 0.20)})
        augmented.append({"question": q, "answer": rng.choice(MID_TEMPLATES),  "target": jitter(0.40, 0.60)})
        augmented.append({"question": q, "answer": rng.choice(HIGH_TEMPLATES), "target": jitter(0.83, 0.97)})

    return augmented
