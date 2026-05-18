"""Train the SoftSkillRegressor.

Pipeline:
  1. Load seed dataset + question pool, augment with quality tiers
  2. Encode (question + " " + answer) via sentence-transformers
  3. StandardScaler on embeddings (saved alongside the model)
  4. Train 200 epochs with early stopping on val loss
  5. Save best weights to weights/best_model_v2.pt and the scaler to
     weights/scaler.npz

Usage (inside the container at startup):
    python -m app.train

Idempotent — re-runs only if best_model_v2.pt is missing or stale.
"""

from __future__ import annotations

import json
import os
import random
from pathlib import Path

import numpy as np
import torch
import torch.nn as nn
import torch.optim as optim
from sklearn.metrics import mean_absolute_error, r2_score
from sklearn.model_selection import train_test_split
from sklearn.preprocessing import StandardScaler
from torch.utils.data import DataLoader, TensorDataset

from app.dataset import augment, load_question_pool, load_seed
from app.model import SoftSkillRegressor

ROOT = Path(__file__).resolve().parent.parent
DATA_DIR = ROOT / "data"
WEIGHTS_DIR = ROOT / "weights"
WEIGHTS_DIR.mkdir(parents=True, exist_ok=True)

BEST_PATH = WEIGHTS_DIR / "best_model_v2.pt"
SCALER_PATH = WEIGHTS_DIR / "scaler.npz"
AUGMENTED_PATH = DATA_DIR / "dataset_augmented.json"

SEED = 42
EMBEDDING_MODEL = os.environ.get("SOFTSKILLS_EMBEDDING_MODEL", "cointegrated/rubert-tiny2")
EPOCHS = int(os.environ.get("SOFTSKILLS_EPOCHS", "200"))
PATIENCE = int(os.environ.get("SOFTSKILLS_PATIENCE", "25"))
BATCH = int(os.environ.get("SOFTSKILLS_BATCH", "32"))
LR = float(os.environ.get("SOFTSKILLS_LR", "8e-4"))


def main() -> None:
    rng = random.Random(SEED)
    torch.manual_seed(SEED)
    np.random.seed(SEED)

    print(f"[train] loading seed from {DATA_DIR}")
    seed = load_seed(DATA_DIR / "dataset_seed.json")
    pool = load_question_pool(DATA_DIR / "questions_pool.json")
    print(f"[train] seed: {len(seed)} samples · pool: {len(pool)} questions")

    augmented = augment(seed, pool, rng)
    print(f"[train] augmented: {len(augmented)} samples")
    with open(AUGMENTED_PATH, "w", encoding="utf-8") as f:
        json.dump(augmented, f, ensure_ascii=False, indent=2)

    # Encode
    print(f"[train] loading embedding model: {EMBEDDING_MODEL}")
    from sentence_transformers import SentenceTransformer  # heavy import — defer
    encoder = SentenceTransformer(EMBEDDING_MODEL)

    texts = [f"{s['question']} {s['answer']}" for s in augmented]
    targets = np.array([[s["target"]] for s in augmented], dtype=np.float32)
    print(f"[train] encoding {len(texts)} texts...")
    embeddings = encoder.encode(texts, batch_size=64, show_progress_bar=False, convert_to_numpy=True)
    embeddings = embeddings.astype(np.float32)
    print(f"[train] embeddings shape: {embeddings.shape}")

    scaler = StandardScaler()
    embeddings_scaled = scaler.fit_transform(embeddings).astype(np.float32)
    np.savez(SCALER_PATH, mean=scaler.mean_, scale=scaler.scale_)
    print(f"[train] scaler saved → {SCALER_PATH}")

    X_train, X_tmp, y_train, y_tmp = train_test_split(
        embeddings_scaled, targets, test_size=0.2, random_state=SEED,
    )
    X_val, X_test, y_val, y_test = train_test_split(
        X_tmp, y_tmp, test_size=0.5, random_state=SEED,
    )
    print(f"[train] split: train={len(X_train)} val={len(X_val)} test={len(X_test)}")

    Xt_train = torch.from_numpy(X_train)
    yt_train = torch.from_numpy(y_train)
    Xt_val = torch.from_numpy(X_val)
    yt_val = torch.from_numpy(y_val)
    Xt_test = torch.from_numpy(X_test)
    yt_test = torch.from_numpy(y_test)

    train_loader = DataLoader(TensorDataset(Xt_train, yt_train), batch_size=BATCH, shuffle=True)
    val_loader = DataLoader(TensorDataset(Xt_val, yt_val), batch_size=BATCH)

    model = SoftSkillRegressor(input_dim=embeddings.shape[1])
    loss_fn = nn.SmoothL1Loss()  # robust to noisy targets
    optimizer = optim.AdamW(model.parameters(), lr=LR, weight_decay=0.01)
    scheduler = optim.lr_scheduler.CosineAnnealingLR(optimizer, T_max=EPOCHS, eta_min=LR * 0.05)

    best_val = float("inf")
    no_improve = 0
    for epoch in range(1, EPOCHS + 1):
        model.train()
        train_loss = 0.0
        for xb, yb in train_loader:
            optimizer.zero_grad()
            pred = model(xb)
            loss = loss_fn(pred, yb)
            loss.backward()
            torch.nn.utils.clip_grad_norm_(model.parameters(), max_norm=1.0)
            optimizer.step()
            train_loss += loss.item()
        train_loss /= max(1, len(train_loader))

        model.eval()
        val_loss = 0.0
        with torch.no_grad():
            for xb, yb in val_loader:
                pred = model(xb)
                val_loss += loss_fn(pred, yb).item()
        val_loss /= max(1, len(val_loader))

        scheduler.step()

        if epoch % 10 == 0:
            print(f"[train] epoch {epoch:3d}/{EPOCHS}  train={train_loss:.4f}  val={val_loss:.4f}  lr={scheduler.get_last_lr()[0]:.6f}")

        if val_loss < best_val - 1e-5:
            best_val = val_loss
            no_improve = 0
            torch.save(model.state_dict(), BEST_PATH)
        else:
            no_improve += 1
            if no_improve >= PATIENCE:
                print(f"[train] early stop at epoch {epoch} (best val={best_val:.4f})")
                break

    # Final test eval
    model.load_state_dict(torch.load(BEST_PATH, map_location="cpu"))
    model.eval()
    with torch.no_grad():
        preds = model(Xt_test).numpy()
    mae = mean_absolute_error(y_test, preds)
    r2 = r2_score(y_test, preds)
    print(f"\n[train] TEST  MAE={mae:.4f}  R²={r2:.4f}  (n={len(y_test)})")
    print(f"[train] weights → {BEST_PATH}")


if __name__ == "__main__":
    main()
