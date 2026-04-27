import json
import os
import re
import statistics
import sys
import time
import urllib.error
import urllib.request

GW = "http://localhost:8000"
AI = "http://localhost:3006/api/v1"
INTERVIEW_BASE = f"{GW}/api/interviews/sessions"
ROLES = ["backend", "frontend", "devops"]
TURNS_PER_ROLE = 8


def post(url, payload, timeout=60, headers=None):
    data = json.dumps(payload).encode("utf-8")
    merged_headers = {"Content-Type": "application/json"}
    if headers:
        merged_headers.update(headers)
    req = urllib.request.Request(url, data=data, headers=merged_headers)
    started = time.time()
    with urllib.request.urlopen(req, timeout=timeout) as r:
        body = r.read().decode("utf-8")
        return r.status, body, (time.time() - started) * 1000


def get(url, timeout=60, headers=None):
    req = urllib.request.Request(url, headers=headers or {})
    with urllib.request.urlopen(req, timeout=timeout) as r:
        return r.status, r.read().decode("utf-8")


def fp(text):
    text = text.lower().strip()
    text = re.sub(r"[^\w\s]+", " ", text)
    text = re.sub(r"\s+", " ", text).strip()
    return text


def p95(values):
    if not values:
        return None
    vals = sorted(values)
    idx = max(0, int(len(vals) * 0.95) - 1)
    return round(vals[idx], 1)


def main():
    run_tag = sys.argv[1] if len(sys.argv) > 1 else os.getenv("SEM_TAG", "before")
    password = "SmokePass123"

    token = None
    auth_error = ""
    for _ in range(6):
        email = f"sem_before_{int(time.time() * 1000)}@example.com"
        username = f"SemBefore{int(time.time() * 1000) % 1000000}"
        try:
            _, reg_body, _ = post(
                f"{GW}/api/auth/register",
                {"email": email, "password": password, "username": username},
                timeout=20,
            )
            reg_json = json.loads(reg_body)
            token = reg_json.get("token") or reg_json.get("access_token")
            if token:
                break
        except urllib.error.HTTPError as exc:
            auth_error = f"register_http_{exc.code}"
        except Exception as exc:
            auth_error = f"register_err_{exc}"

        try:
            _, body, _ = post(
                f"{GW}/api/auth/login",
                {"email": email, "password": password},
                timeout=20,
            )
            login_json = json.loads(body)
            token = login_json.get("token") or login_json.get("access_token")
            if token:
                break
        except urllib.error.HTTPError as exc:
            auth_error = f"login_http_{exc.code}"
        except Exception as exc:
            auth_error = f"login_err_{exc}"

        time.sleep(0.6)

    if not token:
        raise RuntimeError(f"No token in auth responses: {auth_error}")

    auth_headers = {"Authorization": f"Bearer {token}"}
    role_metrics = {}
    all_latencies = []

    for role in ROLES:
        _, sbody, _ = post(
            INTERVIEW_BASE,
            {
                "role": role,
                "level": "senior",
                "durationMinutes": 15,
                "questionLimit": 30,
            },
            timeout=30,
            headers=auth_headers,
        )
        created = json.loads(sbody)
        session_id = (
            created.get("data", {}).get("session_id")
            or created.get("session", {}).get("sessionId")
        )
        if not session_id:
            raise RuntimeError(f"No session for role {role}")

        ai_questions = []
        latencies = []

        for i in range(TURNS_PER_ROLE):
            started = time.time()
            sent = False
            for _ in range(8):
                try:
                    post(
                        f"{INTERVIEW_BASE}/{session_id}/messages",
                        {
                            "sender": "user",
                            "content": f"Ответ {i+1}: объясню подход, риски и метрики для {role}.",
                        },
                        timeout=40,
                        headers=auth_headers,
                    )
                    sent = True
                    break
                except urllib.error.HTTPError as exc:
                    if exc.code == 409:
                        time.sleep(0.5)
                        continue
                    raise
            if not sent:
                raise RuntimeError(f"Failed to send message after retries for role={role}, turn={i+1}")
            latencies.append((time.time() - started) * 1000)

            _, sess_body = get(
                f"{INTERVIEW_BASE}/{session_id}",
                timeout=30,
                headers=auth_headers,
            )
            sess_json = json.loads(sess_body)
            messages = (
                sess_json.get("data", {}).get("messages", [])
                or sess_json.get("session", {}).get("messages", [])
            )
            last_ai = ""
            for msg in reversed(messages):
                if msg.get("sender") == "ai":
                    last_ai = (msg.get("content") or "").strip()
                    break
            if last_ai:
                ai_questions.append(last_ai)

        all_latencies.extend(latencies)

        fingerprints = [fp(q) for q in ai_questions if q.strip()]
        exact_dup = len(fingerprints) - len(set(fingerprints))

        semantic_pairs = 0
        for i in range(len(ai_questions)):
            for j in range(i + 1, len(ai_questions)):
                try:
                    status, cmp_body, _ = post(
                        f"{AI}/embeddings/compare",
                        {"question1": ai_questions[i], "question2": ai_questions[j], "role": role},
                        timeout=30,
                    )
                    if status == 200 and json.loads(cmp_body).get("is_duplicate"):
                        semantic_pairs += 1
                except Exception:
                    pass

        role_metrics[role] = {
            "ai_questions": len(ai_questions),
            "exact_duplicates": exact_dup,
            "semantic_duplicate_pairs": semantic_pairs,
            "latency_ms_p50": round(statistics.median(latencies), 1) if latencies else None,
            "latency_ms_p95": p95(latencies),
        }

    out = {
        "tag": run_tag,
        "roles": role_metrics,
        "global_latency_ms_p50": round(statistics.median(all_latencies), 1) if all_latencies else None,
        "global_latency_ms_p95": p95(all_latencies),
        "turns_total": len(ROLES) * TURNS_PER_ROLE,
    }
    print(json.dumps(out, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
