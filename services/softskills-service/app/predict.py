"""Inference layer for the soft-skills regressor.

Loaded once at FastAPI startup; subsequent calls are ~5-20ms on CPU.
"""

from __future__ import annotations

import json
import logging
import os
import random
from pathlib import Path
from threading import Lock
from typing import List, Optional

import numpy as np
import torch

from app.model import SoftSkillRegressor, load_state_dict_safe

logger = logging.getLogger(__name__)

ROOT = Path(__file__).resolve().parent.parent
WEIGHTS_DIR = ROOT / "weights"
DATA_DIR = ROOT / "data"

# Embedding model handle is global so the (heavy) load happens once.
_encoder = None
_encoder_lock = Lock()

_model: Optional[SoftSkillRegressor] = None
_scaler_mean: Optional[np.ndarray] = None
_scaler_scale: Optional[np.ndarray] = None


def _get_encoder():
    global _encoder
    if _encoder is None:
        with _encoder_lock:
            if _encoder is None:
                from sentence_transformers import SentenceTransformer
                name = os.environ.get("SOFTSKILLS_EMBEDDING_MODEL", "cointegrated/rubert-tiny2")
                logger.info("loading embedding model: %s", name)
                _encoder = SentenceTransformer(name)
    return _encoder


def _resolve_weights_path() -> Path:
    # Prefer the freshly-trained v2 weights; fall back to legacy v1.
    v2 = WEIGHTS_DIR / "best_model_v2.pt"
    if v2.exists():
        return v2
    return WEIGHTS_DIR / "best_model_v1.pt"


def load_model() -> SoftSkillRegressor:
    """Initialise the regressor and load weights / scaler from disk."""
    global _model, _scaler_mean, _scaler_scale
    if _model is not None:
        return _model

    encoder = _get_encoder()
    input_dim = encoder.get_sentence_embedding_dimension()

    model = SoftSkillRegressor(input_dim=input_dim)
    weights_path = _resolve_weights_path()
    if weights_path.exists():
        ok = load_state_dict_safe(model, str(weights_path))
        logger.info("loaded weights from %s (ok=%s)", weights_path, ok)
    else:
        logger.warning("no weights at %s — using untrained model", weights_path)
    model.eval()

    scaler_path = WEIGHTS_DIR / "scaler.npz"
    if scaler_path.exists():
        z = np.load(scaler_path)
        _scaler_mean = z["mean"].astype(np.float32)
        _scaler_scale = z["scale"].astype(np.float32)
        logger.info("loaded scaler from %s", scaler_path)
    else:
        # Fallback: identity scaling. Slightly degrades quality but
        # keeps inference functional when training hasn't run yet.
        _scaler_mean = np.zeros(input_dim, dtype=np.float32)
        _scaler_scale = np.ones(input_dim, dtype=np.float32)
        logger.warning("no scaler at %s — using identity scaling", scaler_path)

    _model = model
    return model


def predict_score(question: str, answer: str) -> float:
    """Return the soft-skill score in 0..100."""
    model = load_model()
    text = f"{question.strip()} {answer.strip()}"
    encoder = _get_encoder()
    emb = encoder.encode([text], convert_to_numpy=True).astype(np.float32)
    emb = (emb - _scaler_mean) / np.where(_scaler_scale == 0, 1.0, _scaler_scale)
    with torch.no_grad():
        out = model(torch.from_numpy(emb)).item()
    return max(0.0, min(100.0, out * 100.0))


# ---------------------------- Question bank ----------------------------

_questions_cache: Optional[List[str]] = None


def get_questions(limit: int = 0) -> List[str]:
    """Return the deterministic question pool loaded from data/."""
    global _questions_cache
    if _questions_cache is None:
        pool: List[str] = []
        pool_path = DATA_DIR / "questions_pool.json"
        if pool_path.exists():
            with open(pool_path, "r", encoding="utf-8") as f:
                raw = json.load(f)
            items = raw.get("questions", raw) if isinstance(raw, dict) else raw
            seen = set()
            for it in items:
                q = it.get("question") if isinstance(it, dict) else str(it)
                if not q:
                    continue
                q = str(q).strip()
                if q and q not in seen:
                    seen.add(q)
                    pool.append(q)
        _questions_cache = pool
    if limit > 0:
        return _questions_cache[:limit]
    return list(_questions_cache)


def sample_questions(n: int, rng: Optional[random.Random] = None) -> List[str]:
    """Return n unique random questions from the pool."""
    pool = get_questions()
    if not pool:
        return []
    r = rng or random.Random()
    if n >= len(pool):
        return list(pool)
    return r.sample(pool, n)


def feedback_for(score: float) -> str:
    """Short human-readable feedback string for a numeric score."""
    s = max(0.0, min(100.0, score))
    if s >= 85:
        return "Сильный ответ: конкретные примеры и измеримые результаты."
    if s >= 70:
        return "Хороший ответ. Можно усилить конкретикой и метриками."
    if s >= 50:
        return "Средний ответ. Не хватает структуры (ситуация → действие → результат)."
    if s >= 30:
        return "Слабый ответ. Добавьте конкретный пример из опыта."
    return "Очень слабый ответ. Опишите ситуацию, действия и результат."
