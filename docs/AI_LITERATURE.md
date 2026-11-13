# Материалы и ИИ

Основной путь не зависит от ИИ: импорт TXT или аудио → черновики → ручные
правильный/ошибочный переводы → сохранение проверенных вариантов → публикация.
ИИ через Chat Completions опционально заполняет поля редактора. Его ответы нельзя
считать достоверным переводом до проверки человеком. Секрет доступен только backend.

Импорт TXT: до 1 МБ, UTF-8, предложения/абзацы, до 500 фрагментов.
Импорт аудио: MP3/WAV/OGG до 20 МБ, WebVTT UTF-8 до 1 МБ, до 500 непересекающихся
реплик с возрастающими таймкодами. Права на исходник и обработку подтверждает редактор.
Текст/расшифровка и исходное аудио сохраняются; координаты относятся к оригиналу.

Маршруты admin:

- POST /admin/literature/import — TXT или поле text, title, license, rightsConfirmed=true.
- POST /admin/audio/import — audio, transcript (WebVTT), title, license, sourceLang, rightsConfirmed=true.
- GET /admin/literature/drafts?langPair=ba-ru — черновики обоих типов источника.
- POST /admin/literature/fragments/{id}/generate?langPair=ba-ru — необязательные ИИ-черновики.
- PUT /admin/literature/fragments/{id}/translations — {languagePair,correct,incorrect}.
- POST /admin/literature/fragments/{id}/publish — {languagePair}.
- POST /admin/literature/fragments/{id}/reject — отклонение черновика.

Опубликованный эталон не является переводом корпуса, подтверждённым участниками:
он только служит исходным предложением и проверенным критерием качества.
