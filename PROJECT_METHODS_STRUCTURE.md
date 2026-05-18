# Структура методов проекта

## FRONTEND - СТРАНИЦЫ И КОМПОНЕНТЫ

### HOME PAGE (Главная страница)

| Элемент управления | Свойства | Назначение |
|---|---|---|
| getAllCourses | Function = home-load-courses | Загружает список всех опубликованных курсов/интервью из БД |
| formatPrice | Function = home-format-price | Конвертирует числовое значение цены в удобный формат (100 → 100 USD) |
| formatStudents | Function = home-format-count | Сокращает большие числа студентов (1500 → 1.5K) |
| navigate | Function = home-link-register | Переприписывает неавторизованного пользователя при клике на курс |
| onStart | Function = home-btn-start | Функция кнопки "Начать обучение", ведущая к регистрации |
| getTopRatedCourses | Function = home-top-rated | Загружает топ-10 курсов по рейтингу и популярности |
| getRecentCourses | Function = home-recent | Загружает последние добавленные курсы (за неделю) |
| onCourseHover | Function = home-preview | Показывает preview-карточку курса при наведении мыши |
| trackEngagement | Function = home-analytics | Отслеживает клики, время на странице и взаимодействие пользователя |

### AUTH PAGE (Авторизация и регистрация)

| Элемент управления | Свойства | Назначение |
|---|---|---|
| registration | Function = auth-registration | Отправляет данные нового пользователя (имя, email, пароль, роль) на сервер |
| handleChange | Function = auth-input-handler | Универсальный обработчик ввода данных в поля формы в реальном времени |
| passwordsMatch | Function = auth-validate-pwd | Проверяет идентичность пароля и его подтверждения перед отправкой |
| canSubmit | Function = auth-validate-form | Валидирует форму (проверка заполненности всех обязательных полей) |
| setSubmitted | Function = auth-confirmation-state | Переключает интерфейс на сообщение о подтверждении почты |
| validateEmail | Function = auth-email-valid | Проверяет корректность формата email адреса |
| validatePassword | Function = auth-pwd-valid | Проверяет требования пароля (8+ символов, буквы, цифры) |
| login | Function = auth-login | Авторизирует пользователя и сохраняет JWT токен в локальное хранилище |
| logout | Function = auth-logout | Удаляет токен, сессию и перенаправляет на главную |
| getAuthStatus | Function = auth-status | Проверяет статус аутентификации текущего пользователя |
| sendVerificationEmail | Function = auth-send-email | Отправляет письмо подтверждения email на адрес пользователя |
| verifyEmailToken | Function = auth-verify-link | Подтверждает email пользователя по ссылке из письма |

### INTERVIEW PAGE (Интерфейс интервью)

| Элемент управления | Свойства | Назначение |
|---|---|---|
| startInterview | Function = interview-start | Инициирует начало интервью с инициализацией таймера |
| recordAnswer | Function = interview-record-answer | Записывает ответ пользователя на вопрос в память сессии |
| captureVideo | Function = interview-capture-video | Захватывает видео поток с веб-камеры кандидата |
| recordScreen | Function = interview-record-screen | Записывает экран рабочего стола (если требуется для кодинг-вопросов) |
| submitInterview | Function = interview-submit | Отправляет завершенное интервью на проверку в AI-сервис |
| pauseInterview | Function = interview-pause | Ставит интервью на паузу без потери прогресса |
| resumeInterview | Function = interview-resume | Продолжает прерванное интервью |
| getCurrentQuestion | Function = interview-get-current | Получает текущий вопрос интервью из списка |
| nextQuestion | Function = interview-next | Переходит к следующему вопросу интервью |
| previousQuestion | Function = interview-prev | Возвращается к предыдущему вопросу |
| getTimeRemaining | Function = interview-timer | Возвращает оставшееся время на интервью в секундах |
| checkCameraPermissions | Function = interview-camera-check | Проверяет разрешения браузера на использование камеры |
| checkMicrophonePermissions | Function = interview-mic-check | Проверяет разрешения на использование микрофона |
| saveProgressLocally | Function = interview-save-local | Сохраняет прогресс в localStorage для восстановления сессии |

### INTERVIEW RESULT PAGE (Результаты интервью)

| Элемент управления | Свойства | Назначение |
|---|---|---|
| getInterviewResults | Function = result-load-data | Загружает результаты завершенного интервью с сервера |
| calculateScore | Function = result-calculate-score | Вычисляет финальный балл кандидата на основе ответов |
| generateFeedback | Function = result-generate-feedback | Генерирует персональный отзыв по результатам через AI |
| displayScoreBreakdown | Function = result-breakdown | Показывает детальное разложение баллов по категориям |
| exportResultsToPDF | Function = result-export-pdf | Экспортирует результаты интервью в PDF документ |
| shareResults | Function = result-share | Делится результатами в соцсетях (LinkedIn, Twitter) |
| compareWithAverage | Function = result-compare-avg | Сравнивает результаты со средним баллом по должности |
| suggestImprovements | Function = result-improvements | Предлагает персональные области для улучшения навыков |

---

### 5️⃣ DASHBOARD (Персональный кабинет)

| Метод | Описание |
|-------|---------|
| **getUserStats()** | Загружает статистику пользователя (интервью, баллы) |
| **getRecentInterviews(limit)** | Получает последние N интервью |
| **getUpcomingInterviews()** | Загружает предстоящие запланированные интервью |
| **calculateCompletionRate()** | Вычисляет процент завершенных интервью |
| **getPerformanceTrend()** | Получает тренд производительности за время |
| **updateProfile(data)** | Обновляет профиль пользователя |
| **changePassword(oldPwd, newPwd)** | Меняет пароль пользователя |
| **enableTwoFactor()** | Включает двухфакторную аутентификацию |
| **getNotifications()** | Загружает уведомления пользователя |
| **markNotificationRead(notifId)** | Отмечает уведомление как прочитанное |

---

### 6️⃣ PROFILE PAGE (Профиль пользователя)

| Метод | Описание |
|-------|---------|
| **getProfileData(userId)** | Загружает данные профиля пользователя |
| **updateProfilePicture(image)** | Загружает новую фотографию профиля |
| **updateBio(text)** | Обновляет биографию пользователя |
| **addSkill(skill)** | Добавляет навык в список умений |
| **removeSkill(skill)** | Удаляет навык из списка |
| **addEducation(record)** | Добавляет запись об образовании |
| **addExperience(record)** | Добавляет запись об опыте работы |
| **getViewCount()** | Получает количество просмотров профиля |
| **getFollowers()** | Получает список подписчиков |
| **follow(userId)** | Подписывается на пользователя |

---

### 7️⃣ RESUME PAGE (Резюме)

| Метод | Описание |
|-------|---------|
| **uploadResume(file)** | Загружает файл резюме (PDF/DOCX) |
| **parseResume(fileData)** | Парсит текст из резюме (OCR) |
| **extractSkills(resumeText)** | Извлекает навыки из текста резюме |
| **validateResumeFormat()** | Проверяет формат и расширение файла |
| **generateAIResume()** | Генерирует резюме через ИИ на основе инфо профиля |
| **downloadResume(format)** | Скачивает резюме в выбранном формате |
| **viewResumePreview()** | Показывает превью резюме |
| **trackResumeViews()** | Отслеживает, кто просматривал резюме |

---

### 8️⃣ REPORTS PAGE (Отчеты и аналитика)

| Метод | Описание |
|-------|---------|
| **getReportsList()** | Загружает список всех доступных отчетов |
| **generateReport(type, dateRange)** | Генерирует отчет для выбранного периода |
| **downloadReport(reportId, format)** | Скачивает отчет (PDF, Excel, CSV) |
| **getReportDetails(reportId)** | Загружает детали конкретного отчета |
| **filterReportsByDate(startDate, endDate)** | Фильтрует отчеты по дате |
| **compareReports(reportId1, reportId2)** | Сравнивает два отчета |
| **scheduleReportEmail(frequency)** | Настраивает автоматическую отправку отчетов |
| **getPerformanceMetrics()** | Получает метрики производительности |

---

### 9️⃣ ADMIN PAGE (Админ панель)

| Метод | Описание |
|-------|---------|
| **getAllUsers()** | Загружает список всех пользователей |
| **blockUser(userId)** | Блокирует пользователя |
| **unblockUser(userId)** | Разблокирует пользователя |
| **deleteUser(userId)** | Удаляет пользователя и его данные |
| **getSystemStats()** | Получает статистику системы |
| **getServerHealth()** | Проверяет здоровье серверов |
| **viewAuditLog()** | Просматривает логи действий |
| **manageSuspiciousActivity(userId)** | Управляет подозрительной активностью |
| **sendBroadcastMessage(message)** | Отправляет массовое сообщение пользователям |
| **exportSystemReport()** | Экспортирует полный отчет системы |

---

## BACKEND - СЕРВИСЫ

### 🔐 USER-SERVICE (Управление пользователями)

| Метод | Описание |
|-------|---------|
| **CreateUser(data)** | Создает нового пользователя с валидацией |
| **GetUser(userId)** | Получает информацию пользователя по ID |
| **UpdateUser(userId, updates)** | Обновляет данные пользователя |
| **DeleteUser(userId)** | Удаляет пользователя из системы |
| **AuthenticateUser(email, pwd)** | Аутентифицирует пользователя, возвращает JWT |
| **VerifyEmail(token)** | Подтверждает email пользователя |
| **ResetPassword(email)** | Отправляет ссылку сброса пароля |
| **ChangePassword(userId, oldPwd, newPwd)** | Меняет пароль пользователя |
| **CheckUserExists(email)** | Проверяет существование пользователя |
| **GetUserRole(userId)** | Получает роль пользователя (candidate/interviewer/admin) |

---

### 📋 INTERVIEW-SERVICE (Управление интервью)

| Метод | Описание |
|-------|---------|
| **CreateInterview(data)** | Создает новое интервью с вопросами |
| **GetInterview(interviewId)** | Загружает информацию об интервью |
| **StartInterview(interviewId, userId)** | Начинает интервью для пользователя |
| **AssignQuestion(interviewId, questionId)** | Назначает вопрос интервью |
| **SubmitAnswer(interviewId, questionId, answer)** | Сохраняет ответ на вопрос |
| **CompleteInterview(interviewId)** | Завершает интервью |
| **PauseInterview(interviewId)** | Ставит интервью на паузу |
| **ResumeInterview(interviewId)** | Возобновляет интервью |
| **SaveInterviewProgress(interviewId, data)** | Сохраняет прогресс интервью |
| **GetInterviewQuestions(interviewId)** | Получает список вопросов интервью |
| **ShuffleQuestions(interviewId)** | Перемешивает порядок вопросов |
| **WebSocketBroadcast(sessionId, event)** | Отправляет события через WebSocket |

---

### ⭐ SCORING-SERVICE (Оценка результатов)

| Метод | Описание |
|-------|---------|
| **CalculateScore(interviewId, answers)** | Вычисляет финальный балл |
| **EvaluateAnswer(questionId, answer)** | Оценивает отдельный ответ |
| **CompareDifficultyLevel(score, difficulty)** | Сравнивает балл со сложностью |
| **GenerateAIFeedback(answers)** | Генерирует отзыв через ИИ-модель |
| **RankCandidate(userId)** | Ранжирует кандидата среди остальных |
| **IdentifyStrengths(answers)** | Определяет сильные стороны кандидата |
| **IdentifyWeaknesses(answers)** | Определяет слабые стороны кандидата |
| **CompareWithBenchmark(score)** | Сравнивает с бенчмарками |

---

### 💻 CODE-EXECUTOR-SERVICE (Исполнение кода)

| Метод | Описание |
|-------|---------|
| **ExecuteCode(language, code, input)** | Выполняет code и возвращает output |
| **ValidateCodeSyntax(code, language)** | Проверяет синтаксис кода |
| **RunTests(code, testCases)** | Запускает тесты для кода |
| **DetectCodeIssues(code)** | Определяет проблемы в коде (стиль, производительность) |
| **CompileCode(code, language)** | Компилирует код перед выполнением |
| **SetTimeout(execution, maxTime)** | Устанавливает таймаут выполнения |
| **CaptureOutput(execution)** | Захватывает вывод программы |
| **GetExecutionMetrics(execution)** | Получает метрики выполнения (время, память) |

---

### 📊 ANALYTICS-SERVICE (Аналитика)

| Метод | Описание |
|-------|---------|
| **TrackUserAction(userId, action)** | Отслеживает действие пользователя |
| **GetUserMetrics(userId, period)** | Получает метрики пользователя за период |
| **GetSystemMetrics()** | Получает общие метрики системы |
| **GenerateHeatmap()** | Генерирует тепловую карту активности |
| **GetConversionRate()** | Получает коэффициент конверсии |
| **AnalyzeTrends(metric, timeRange)** | Анализирует тренды метрики |
| **CalculateEngagement(userId)** | Вычисляет уровень вовлеченности |
| **PredictChurn(userId)** | Предсказывает вероятность ухода пользователя |

---

### 📄 REPORT-SERVICE (Генерация отчетов)

| Метод | Описание |
|-------|---------|
| **GenerateReport(type, data)** | Генерирует отчет выбранного типа |
| **ExportToPDF(reportData)** | Экспортирует отчет в PDF |
| **ExportToExcel(reportData)** | Экспортирует отчет в Excel |
| **CreateScheduledReport(schedule, recipients)** | Создает запланированный отчет |
| **SendReportEmail(reportId, recipients)** | Отправляет отчет на email |
| **ViewReportHistory()** | Получает историю сгенерированных отчетов |
| **CustomizeReportTemplate(template)** | Кастомизирует шаблон отчета |
| **CompareReports(id1, id2)** | Сравнивает два отчета |

---

### 🔑 GITHUB-SERVICE (Интеграция GitHub)

| Метод | Описание |
|-------|---------|
| **AuthenticateWithGithub(code)** | Аутентифицирует пользователя через GitHub OAuth |
| **GetRepositories(userId)** | Получает список репозиториев пользователя |
| **GetRepositoryStats(repoId)** | Получает статистику репозитория |
| **AnalyzeCodeQuality(repoId)** | Анализирует качество кода в репозитории |
| **GetContributionGraph(userId)** | Получает граф контрибьюций |
| **FetchCommitHistory(repoId)** | Получает историю коммитов |
| **LinkGithubAccount(userId, githubId)** | Привязывает GitHub аккаунт к профилю |
| **UnlinkGithubAccount(userId)** | Отвязывает GitHub аккаунт |

---

### 📩 NOTIFICATION-SERVICE (Уведомления)

| Метод | Описание |
|-------|---------|
| **SendEmail(recipient, subject, body)** | Отправляет email |
| **SendSMS(phone, message)** | Отправляет SMS |
| **SendPushNotification(userId, title, body)** | Отправляет пуш-уведомление |
| **CreateNotification(userId, data)** | Создает в-app уведомление |
| **GetUserNotifications(userId)** | Получает уведомления пользователя |
| **MarkAsRead(notificationId)** | Отмечает уведомление как прочитанное |
| **DeleteNotification(notificationId)** | Удаляет уведомление |
| **BulkSendNotifications(users, message)** | Отправляет массовое уведомление |

---

### 📄 RESUME-SERVICE (Работа с резюме)

| Метод | Описание |
|-------|---------|
| **UploadResume(userId, file)** | Загружает резюме пользователя |
| **ParseResume(fileData)** | Парсит текст из резюме (OCR/PDF) |
| **ExtractSkills(resumeText)** | Извлекает навыки из текста |
| **ValidateResume(fileFormat)** | Валидирует формат резюме |
| **GenerateAIResume(userProfile)** | Генерирует резюме на основе профиля |
| **GetResume(userId)** | Получает резюме пользователя |
| **UpdateResume(userId, updates)** | Обновляет резюме |
| **DeleteResume(userId)** | Удаляет резюме |

---

### 🤖 AI-SERVICE (Искусственный интеллект)

| Метод | Описание |
|-------|---------|
| **AnalyzeAnswer(question, answer, context)** | Анализирует ответ кандидата с помощью ИИ |
| **GenerateFeedback(answers, model)** | Генерирует отзыв через ИИ-модель |
| **PredictScore(answersFeatures)** | Предсказывает балл на основе ML-модели |
| **RecommendSkills(userProfile)** | Рекомендует навыки для улучшения |
| **DetectCheat(answerPattern, submission)** | Определяет попытки обмана/плагиата |
| **OptimizeQuestions(difficulty)** | Оптимизирует сложность вопросов |
| **AnalyzeTone(answer)** | Анализирует тон и эмоции в ответе |

---

### 👮 ADMIN-SERVICE (Администрирование)

| Метод | Описание |
|-------|---------|
| **GetAllUsers()** | Получает список всех пользователей |
| **BlockUser(userId, reason)** | Блокирует пользователя |
| **UnblockUser(userId)** | Разблокирует пользователя |
| **DeleteUser(userId)** | Удаляет пользователя и его данные |
| **GetSystemStats()** | Получает статистику системы |
| **ViewAuditLog(filter)** | Просматривает логи действий |
| **ManageContent(contentId, action)** | Управляет контентом (модерация) |
| **SendSystemNotification(users, message)** | Отправляет системное уведомление |

---

### 🌐 API-GATEWAY (Маршрутизация запросов)

| Метод | Описание |
|-------|---------|
| **RouteRequest(endpoint, method, data)** | Маршрутизирует запрос на нужный сервис |
| **ValidateToken(token)** | Валидирует JWT токен |
| **CheckRateLimit(userId)** | Проверяет rate limiting |
| **LogRequest(endpoint, userId, status)** | Логирует все запросы |
| **HandleError(statusCode, error)** | Обрабатывает ошибки и возвращает response |
| **CacheResponse(endpoint, ttl)** | Кэширует responses |
| **CompressResponse(data)** | Сжимает response для оптимизации |
