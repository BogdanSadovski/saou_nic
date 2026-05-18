# Unit Тесты проекта

## Запуск тестов

### Запуск (рекомендуется подробный вывод)

Из корня проекта выполните:

```bash
cd /Users/bogdan./Documents/учеба/дипломчик/real_ass
pip install -r tests/sample_unit_tests/requirements.txt
# Запуск с выводом print() и подробностями
pytest tests/sample_unit_tests -s -v
```

Или если вы находитесь уже в папке тестов:

```bash
cd tests/sample_unit_tests
pip install -r requirements.txt
pytest . -s -v
```

### Запуск конкретного теста

```bash
# Запуск одного теста
pytest tests/sample_unit_tests/test_realistic.py::TestAPIResponseValidation::test_parse_sample_api_response -v

# Запустить все тесты класса
pytest tests/sample_unit_tests/test_realistic.py::TestAPIResponseValidation -v
```

### Запуск с кратким выводом

```bash
pytest tests/sample_unit_tests -q
```

---

## Описание тестов

### Текущие тесты (5 шт.)

- **test_api_response_structure** — проверяет базовую структуру API-ответа из `testdata/sample_api_response.json` (поле `status`, `data.id`, `data.name`) и печатает весь payload.
- **test_api_response_items_nonempty** — проверяет, что `data.items` — непустой список; печатает первые заголовки элементов.
- **test_frontend_package_json_fields** — проверяет, что в `frontend/package.json` присутствуют поля `name` и `version` и печатает их значения.
- **test_frontend_routing_pages_exist** — проверяет, что в `frontend/src/pages` есть хотя бы одна из ключевых страниц (Home/Interview/Profile) и печатает найденные страницы.
- **test_services_directory_and_configs** — проверяет, что папка `services` существует, перечисляет подпапки и для каждой печатает наличие `config.yaml`/`config.yml`.

### 🔌 TestAPIResponseValidation (Валидация API ответов)
- **test_parse_sample_api_response** — парсит sample API response и проверяет status
- **test_api_response_has_data_object** — проверяет структуру data объекта (id, name)
- **test_api_response_items_list_not_empty** — проверяет, что items список не пустой

### 📊 TestDataProcessing (Обработка данных)
- **test_filter_items_by_id** — фильтрация items по id
- **test_extract_titles_from_items** — извлечение названий из items

### 🌐 TestMockHTTP (Mock HTTP запросы)
- **test_mock_api_call_success** — проверяет mock HTTP GET запрос
- **test_mock_data_validation** — валидация mock данных

### 🏗️ TestProjectComponents (Компоненты проекта)
- **test_services_directory_has_subdirectories** — проверяет, что services содержит подпапки
- **test_frontend_source_structure** — проверяет структуру frontend/src (components, pages, hooks, utils)

---

## Что тестируется

✅ Структура проекта (файлы, директории)  
✅ Парсинг JSON (API responses)  
✅ Валидация данных (schema, types)  
✅ Обработка данных (filtering, extraction)  
✅ Mock HTTP запросы  
✅ Бизнес-логика (фильтрация items, извлечение titles)  

---

## Требования

- Python 3.8+
- pytest >= 7.0

## Установка зависимостей

```bash
pip install -r tests/sample_unit_tests/requirements.txt
```

или вручную:

```bash
pip install pytest
```
