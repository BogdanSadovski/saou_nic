"""FastAPI service exposing the soft-skills regressor.

Endpoints
---------
GET  /health                  → liveness probe
GET  /api/v1/questions?n=5    → sample n questions from the bank
POST /api/v1/score            → {question, answer} → {score, feedback}
POST /api/v1/score_batch      → batch of pairs → list of scores
POST /api/v1/session/score    → score a whole interview session, returns
                                per-turn detail + averaged report
"""

from __future__ import annotations

import logging
import os
import random
from typing import List, Optional

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from pydantic import BaseModel, Field

from app.predict import (
    feedback_for,
    load_model,
    predict_score,
    sample_questions,
)

logging.basicConfig(
    # Accept "info"/"INFO" both ways — Python's basicConfig is
    # case-sensitive and crash-loops on the lowercase form, which is
    # the default LOG_LEVEL in our docker-compose for Go services.
    level=os.environ.get("LOG_LEVEL", "INFO").upper(),
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("softskills")

app = FastAPI(title="RealSync · Soft-Skills Scorer", version="1.0.0")

# Permissive CORS — gateway sits in front, so direct browser hits are
# unusual but harmless.
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.on_event("startup")
async def _warmup() -> None:
    # Eager-load encoder + model so the first /score doesn't pay the
    # ~3-5s cold-start tax mid-interview.
    try:
        load_model()
        logger.info("soft-skills model warmed up successfully")
    except Exception as exc:  # noqa: BLE001
        logger.warning("warmup failed (will retry lazily): %s", exc)


@app.get("/health")
async def health() -> dict:
    return {"status": "ok", "service": "softskills-service"}


# ─── Schemas ───────────────────────────────────────────────────────────

class ScoreRequest(BaseModel):
    question: str = Field(..., min_length=1, max_length=2000)
    answer: str = Field(..., min_length=1, max_length=20000)


class ScoreResponse(BaseModel):
    score: float
    feedback: str
    verdict: str  # correct | partial | wrong | skipped


class ScoreBatchItem(BaseModel):
    question: str
    answer: str


class ScoreBatchRequest(BaseModel):
    items: List[ScoreBatchItem]


class ScoreBatchEntry(BaseModel):
    question: str
    answer: str
    score: float
    feedback: str
    verdict: str


class ScoreBatchResponse(BaseModel):
    results: List[ScoreBatchEntry]
    average_score: float
    overall_feedback: str


class QuestionsResponse(BaseModel):
    questions: List[str]


class SessionTurn(BaseModel):
    question: str
    answer: str


class SessionScoreRequest(BaseModel):
    turns: List[SessionTurn]


class SessionScoreResponse(BaseModel):
    overall_score: float
    overall_feedback: str
    per_turn: List[ScoreBatchEntry]
    strengths: List[str]
    weaknesses: List[str]
    recommendations: List[str]


# ─── Helpers ───────────────────────────────────────────────────────────

def _verdict_for(score: float, answer: str) -> str:
    a = (answer or "").strip()
    if not a or len(a) < 4:
        return "skipped"
    if score >= 75:
        return "correct"
    if score >= 50:
        return "partial"
    return "wrong"


# ─── Routes ────────────────────────────────────────────────────────────

@app.get("/api/v1/questions", response_model=QuestionsResponse)
async def get_questions(n: int = 5, seed: Optional[int] = None) -> QuestionsResponse:
    if n <= 0 or n > 100:
        raise HTTPException(status_code=400, detail="n must be in [1, 100]")
    rng = random.Random(seed) if seed is not None else None
    return QuestionsResponse(questions=sample_questions(n, rng))


@app.post("/api/v1/score", response_model=ScoreResponse)
async def score(req: ScoreRequest) -> ScoreResponse:
    s = predict_score(req.question, req.answer)
    return ScoreResponse(score=s, feedback=feedback_for(s), verdict=_verdict_for(s, req.answer))


@app.post("/api/v1/score_batch", response_model=ScoreBatchResponse)
async def score_batch(req: ScoreBatchRequest) -> ScoreBatchResponse:
    if not req.items:
        raise HTTPException(status_code=400, detail="items must be non-empty")
    results: List[ScoreBatchEntry] = []
    total = 0.0
    for it in req.items:
        s = predict_score(it.question, it.answer)
        results.append(
            ScoreBatchEntry(
                question=it.question,
                answer=it.answer,
                score=s,
                feedback=feedback_for(s),
                verdict=_verdict_for(s, it.answer),
            )
        )
        total += s
    avg = total / len(results)
    return ScoreBatchResponse(
        results=results,
        average_score=avg,
        overall_feedback=feedback_for(avg),
    )


@app.post("/api/v1/session/score", response_model=SessionScoreResponse)
async def session_score(req: SessionScoreRequest) -> SessionScoreResponse:
    """Score an entire soft-skills session and produce a report."""
    if not req.turns:
        raise HTTPException(status_code=400, detail="turns must be non-empty")

    per_turn: List[ScoreBatchEntry] = []
    total = 0.0
    for t in req.turns:
        s = predict_score(t.question, t.answer)
        per_turn.append(
            ScoreBatchEntry(
                question=t.question,
                answer=t.answer,
                score=s,
                feedback=feedback_for(s),
                verdict=_verdict_for(s, t.answer),
            )
        )
        total += s
    avg = total / max(1, len(per_turn))

    sorted_by_score = sorted(per_turn, key=lambda e: e.score, reverse=True)
    strengths = []
    weaknesses = []
    for e in sorted_by_score[:3]:
        if e.score >= 65:
            strengths.append(f"Сильно: «{e.question[:60]}…» ({e.score:.0f}%)")
    for e in sorted_by_score[::-1][:3]:
        if e.score < 60:
            weaknesses.append(f"Слабо: «{e.question[:60]}…» ({e.score:.0f}%)")

    if not strengths:
        strengths.append("Базовый уровень коммуникации — структура ответов прослеживается.")

    recommendations = [
        "Используйте формулу STAR: Situation → Task → Action → Result.",
        "Подкрепляйте ответы конкретными цифрами (сроки, размер команды, метрики).",
        "Заканчивайте каждый ответ итогом — чему вы научились / какой результат получили.",
    ]
    if avg < 50:
        recommendations.insert(0, "Тренируйтесь рассказывать о конкретных ситуациях, а не давать общие правила.")

    return SessionScoreResponse(
        overall_score=avg,
        overall_feedback=feedback_for(avg),
        per_turn=per_turn,
        strengths=strengths,
        weaknesses=weaknesses or ["Явных слабых сторон не обнаружено."],
        recommendations=recommendations,
    )
