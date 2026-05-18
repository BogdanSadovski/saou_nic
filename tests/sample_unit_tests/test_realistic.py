"""Набор из 5 тестов: проверка API-ответа, проверка non-empty items,
проверка frontend package.json, проверка роутинга (страницы) и проверка services.

Каждый тест печатает подробную информацию в консоль. Запускайте с флагом `-s -v`,
чтобы увидеть вывод print().
"""

import json
from pathlib import Path
import urllib.parse


PROJECT_ROOT = Path(__file__).resolve().parents[2]
TESTDATA = Path(__file__).resolve().parent / "testdata" / "sample_api_response.json"


def _read_json(path: Path):
    text = path.read_text(encoding="utf-8")
    return json.loads(text)


def test_api_response_structure():
    """Проверяет базовую структуру API-ответа: status, data.id, data.name."""
    assert TESTDATA.exists(), f"Testdata not found: {TESTDATA}"
    payload = _read_json(TESTDATA)
    print('\n[DEBUG] Loaded payload:', json.dumps(payload, ensure_ascii=False))

    assert isinstance(payload, dict), "Payload must be JSON object"
    status = payload.get("status")
    print(f"[DEBUG] status = {status}")
    assert status in ("ok", "error"), "status must be 'ok' or 'error'"

    data = payload.get("data")
    print(f"[DEBUG] data keys = {list(data.keys()) if isinstance(data, dict) else data}")
    assert isinstance(data, dict), "data must be an object"
    assert "id" in data and data["id"] is not None, "data.id is required"
    assert "name" in data and isinstance(data["name"], str), "data.name must be a string"


def test_api_response_items_nonempty():
    """Проверяет, что в data.items есть элементы и печатает их кратко."""
    payload = _read_json(TESTDATA)
    items = payload.get("data", {}).get("items")
    print(f"\n[DEBUG] items: {items}")
    assert isinstance(items, list), "items must be a list"
    assert len(items) > 0, "items must not be empty"

    # Печать первых 3 заголовков для диагностики
    titles = [it.get("title") for it in items[:3]]
    print(f"[DEBUG] first titles = {titles}")


def test_frontend_package_json_fields():
    """Проверяет frontend/package.json: presence of name and version и печатает их."""
    pkg = PROJECT_ROOT / "frontend" / "package.json"
    assert pkg.exists(), f"frontend/package.json not found at {pkg}"
    data = _read_json(pkg)
    name = data.get("name")
    version = data.get("version")
    print(f"\n[DEBUG] frontend package name={name} version={version}")
    assert isinstance(name, str) and name.strip(), "package.json.name is required"
    assert isinstance(version, str) and version.strip(), "package.json.version is required"


def test_frontend_routing_pages_exist():
    """Проверяет наличие ключевых frontend страниц/роутов (Home, Interview, Profile)."""
    pages_dir = PROJECT_ROOT / "frontend" / "src" / "pages"
    print(f"\n[DEBUG] checking pages in {pages_dir}")
    assert pages_dir.is_dir(), f"frontend pages dir not found at {pages_dir}"

    # Список файлов/папок в pages
    entries = [p.name for p in pages_dir.iterdir()]
    print(f"[DEBUG] pages entries: {entries}")

    expect = {"Home", "Interview", "Profile"}
    # Некоторые проекты используют lowercase или index files; приводим к набору по названию
    found = set([e.split('.')[0].split('-')[0].capitalize() for e in entries])
    print(f"[DEBUG] normalized found pages: {found}")
    assert any(x in found for x in expect), f"None of expected pages {expect} found in {found}"


def test_services_directory_and_configs():
    """Проверяет, что папка services существует и у сервисов есть config.yaml (если есть).
    Печатает найденные сервисы и их конфиги.
    """
    services = PROJECT_ROOT / "services"
    print(f"\n[DEBUG] services dir = {services}")
    assert services.is_dir(), "services directory must exist"

    subdirs = [p for p in services.iterdir() if p.is_dir()]
    print(f"[DEBUG] found service dirs: {[p.name for p in subdirs]}")
    assert len(subdirs) > 0, "should contain at least one service directory"

    # Для каждого сервиса проверяем наличие config.yaml или config.yml
    for svc in subdirs[:10]:
        cfg1 = svc / "config.yaml"
        cfg2 = svc / "config.yml"
        exists = cfg1.exists() or cfg2.exists()
        print(f"[DEBUG] service={svc.name} config_found={exists} path1={cfg1 if cfg1.exists() else cfg2 if cfg2.exists() else 'none'}")
        # не требуем наличие конфига для всех сервисов, но выводим информацию

