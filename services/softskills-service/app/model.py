"""Soft-skills regression model.

The architecture is an improvement on the original `FlatRegressor`:

  - Wider first hidden layer (192 vs 128)
  - Residual connection between first and second hidden blocks
  - LayerNorm instead of BatchNorm (more stable for small batches and
    eval-mode inference one-by-one)
  - GELU activation instead of ReLU (smoother gradient, generally
    better for text-feature heads)
  - Configurable dropout schedule

Input is a sentence-transformer embedding (default 312-dim from
`cointegrated/rubert-tiny2`). Output is a sigmoid-activated scalar in
[0, 1] representing the soft-skill score, which we expose as 0-100 %.
"""

from __future__ import annotations

import torch
import torch.nn as nn


class SoftSkillRegressor(nn.Module):
    def __init__(
        self,
        input_dim: int = 312,
        hidden1: int = 192,
        hidden2: int = 96,
        dropout1: float = 0.35,
        dropout2: float = 0.20,
    ) -> None:
        super().__init__()
        self.input_dim = input_dim

        self.fc1 = nn.Linear(input_dim, hidden1)
        self.norm1 = nn.LayerNorm(hidden1)
        self.act1 = nn.GELU()
        self.drop1 = nn.Dropout(dropout1)

        self.fc2 = nn.Linear(hidden1, hidden2)
        self.norm2 = nn.LayerNorm(hidden2)
        self.act2 = nn.GELU()
        self.drop2 = nn.Dropout(dropout2)

        # Residual projection so we can sum hidden1 -> hidden2 spaces.
        self.res_proj = nn.Linear(hidden1, hidden2, bias=False)

        self.fc_out = nn.Linear(hidden2, 1)
        self.out_act = nn.Sigmoid()

        self._init_weights()

    def _init_weights(self) -> None:
        for m in self.modules():
            if isinstance(m, nn.Linear):
                nn.init.kaiming_normal_(m.weight, nonlinearity="linear")
                if m.bias is not None:
                    nn.init.zeros_(m.bias)

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        h1 = self.drop1(self.act1(self.norm1(self.fc1(x))))
        h2 = self.act2(self.norm2(self.fc2(h1) + self.res_proj(h1)))
        h2 = self.drop2(h2)
        return self.out_act(self.fc_out(h2))


def load_state_dict_safe(model: nn.Module, path: str, map_location: str = "cpu") -> bool:
    """Load weights into the model, tolerating partial mismatches.

    Returns True if weights were loaded (even partially), False if
    nothing matched. Used at startup to gracefully accept legacy
    `FlatRegressor` checkpoints — common layers (fc1, fc2, fc_out) will
    load, the new residual/layernorm pieces fall back to init.
    """
    try:
        state = torch.load(path, map_location=map_location)
    except Exception:
        return False
    if not isinstance(state, dict):
        return False
    own = model.state_dict()
    loaded = 0
    for k, v in state.items():
        if k in own and own[k].shape == v.shape:
            own[k] = v
            loaded += 1
    model.load_state_dict(own)
    return loaded > 0
