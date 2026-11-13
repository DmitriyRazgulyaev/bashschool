#!/usr/bin/env python3
"""Exercise the API against a disposable database. Never use a production URL.
Start a clean API with ADMIN_EMAIL=admin@example.org and
ADMIN_PASSWORD=review-password-123, then run python3 scripts/acceptance.py.
"""
import http.client
import io
import json
import os
import uuid
import wave
from urllib.parse import urlsplit

base = urlsplit(os.environ.get('API_URL', 'http://127.0.0.1:8080/api/v1'))
checks = []

def request(method, path, body=None, token=None, expected=200, headers=None):
    headers = dict(headers or {})
    if token:
        headers['Authorization'] = 'Bearer ' + token
    if isinstance(body, dict):
        body = json.dumps(body).encode()
        headers['Content-Type'] = 'application/json'
    connection = http.client.HTTPConnection(base.hostname, base.port, timeout=30)
    connection.request(method, base.path + path, body, headers)
    response = connection.getresponse()
    data = response.read()
    assert response.status == expected, (method, path, response.status, data.decode(errors='replace'))
    connection.close()
    if response.getheader('Content-Type', '').startswith('application/json'):
        return json.loads(data)
    return data

def passed(name):
    checks.append(name)
    print('PASS', name)

def multipart(fields, filename, audio):
    boundary = uuid.uuid4().hex
    parts = []
    for key, value in fields.items():
        parts.append(f'--{boundary}\r\nContent-Disposition: form-data; name="{key}"\r\n\r\n{value}\r\n'.encode())
    parts += [f'--{boundary}\r\nContent-Disposition: form-data; name="audio"; filename="{filename}"\r\nContent-Type: audio/wav\r\n\r\n'.encode(), audio, f'\r\n--{boundary}--\r\n'.encode()]
    return b''.join(parts), {'Content-Type': 'multipart/form-data; boundary=' + boundary}

missing = request('POST', '/auth/login', {'email': 'missing@example.org', 'password': 'incorrect-password'}, expected=404)
assert missing['error']['code'] == 'EMAIL_NOT_FOUND'
passed('missing email returns explicit UI message')
admin = request('POST', '/auth/login', {'email': 'admin@example.org', 'password': 'review-password-123'})['accessToken']
request('POST', '/auth/login', {'email': 'admin@example.org', 'password': 'incorrect-password'}, expected=401)
request('POST', '/admin/language-pairs', {'sourceLang': 'ba', 'targetLang': 'en', 'name': 'Acceptance English'}, admin, 201)
users = []
for i in range(7):
    result = request('POST', '/auth/register', {'email': f'acceptance{i}@example.org', 'password': 'review-password-123', 'displayName': f'Ученик {i}', 'languagePair': 'ba-en'}, expected=201)
    users.append(result)
# The original pair must work on the first startup, before any restart.
seed = request('GET', '/tasks/daily?langPair=ba-ru', token=users[0]['accessToken'])
assert seed['sourceText'] and seed['suggestedTranslation']
passed('first startup creates usable demonstration tasks')
material = request('POST', '/materials', {'title': 'Acceptance text', 'type': 'text', 'license': 'Own test text'}, admin, 201)
fragment = request('POST', '/fragments', {'materialId': material['id'], 'textBa': 'Һаумыһығыҙ!', 'sourceLang': 'ba', 'chapter': '1'}, admin, 201)['id']
request('PUT', f'/admin/literature/fragments/{fragment}/translations', {'languagePair': 'ba-en', 'correct': 'Hello!', 'incorrect': 'Goodbye!'}, admin)
request('POST', f'/admin/literature/fragments/{fragment}/publish', {'languagePair': 'ba-en'}, admin)
passed('manual publication works without an AI key')
for i, user in enumerate(users[:5]):
    token = user['accessToken']
    task = request('GET', '/tasks/daily?langPair=ba-en', token=token)
    assert task['fragmentId'] == fragment
    payload = {'answerType': 'correction', 'translation': 'Hello!' if i < 3 else 'Hi!', 'idempotencyKey': str(uuid.uuid4())}
    result = request('POST', f"/tasks/{task['taskId']}/answers", payload, token)
    duplicate = request('POST', f"/tasks/{task['taskId']}/answers", payload, token)
    assert duplicate['alreadyProcessed'] and duplicate['totalXp'] == result['totalXp']
    assert duplicate['xpEarned'] == result['xpEarned'] and duplicate['questBonusXp'] == result['questBonusXp']
    assert result['questBonusXp'] == 25
passed('answers, daily rewards and idempotency')
assert not request('GET', '/translations/export?langPair=ba-en', token=admin)
candidates = request('GET', f'/fragments/{fragment}/candidates?langPair=ba-en', token=admin)
ids = {candidate['translation']: candidate['id'] for candidate in candidates}
request('POST', f'/fragments/{fragment}/equivalences', {'candidateId': ids['Hi!'], 'canonicalId': ids['Hello!']}, admin)
translations = request('GET', '/translations/export?langPair=ba-en', token=admin)
assert len(translations) == 1 and translations[0]['confirmed'] == 5
assert set(translations[0]) == {'fragment_id', 'text_ba', 'lang_pair', 'translation', 'source', 'confirmed', 'status'}
assert request('GET', '/translations/export?langPair=ba-en&format=csv', token=admin).startswith(b'fragment_id,text_ba,lang_pair,translation,source,confirmed,status')
passed('synonym grouping recounts unique voters and exports the required schema')
for user in users[5:]:
    token = user['accessToken']
    task = request('GET', '/tasks/daily?langPair=ba-en', token=token)
    request('POST', f"/tasks/{task['taskId']}/answers", {'answerType': 'correction', 'translation': 'Goodbye!', 'idempotencyKey': str(uuid.uuid4())}, token)
assert not request('GET', '/translations/export?langPair=ba-en', token=admin)
cases = request('GET', '/moderation/cases', token=admin)
case = next(case for case in cases if case['fragmentId'] == fragment)
request('POST', f"/moderation/cases/{case['id']}/resolve", {'candidateId': ids['Hello!']}, admin)
assert request('GET', '/translations/export?langPair=ba-en', token=admin)
passed('lost consensus removes a translation from export; expert resolution restores it')
# Role changes must be effective for the existing token.
teacher = users[0]
request('POST', f"/users/{teacher['user']['id']}/role", {'role': 'teacher'}, admin)
schools = request('GET', '/schools', token=admin)
classroom = request('POST', '/classes', {'name': 'Acceptance class', 'schoolId': schools[0]['id']}, teacher['accessToken'], 201)
student = request('POST', f"/classes/{classroom['id']}/register-student", {'email': 'class-student@example.org', 'password': 'review-password-123', 'displayName': 'Новый ученик'}, teacher['accessToken'], 201)
report = request('GET', f"/classes/{classroom['id']}/reports", token=teacher['accessToken'])
assert report['studentsCount'] == 1
request('POST', f"/classes/{classroom['id']}/register-student", {'email': 'other@example.org', 'password': 'review-password-123', 'displayName': 'Другой'}, users[1]['accessToken'], 403)
passed('teacher registration, class report and role permissions')
for scope in ['global', 'school', 'class']:
    for period in ['daily', 'weekly', 'monthly', 'yearly', 'all']:
        for metric in ['xp', 'activity', 'vocabulary', 'corrections', 'errors']:
            request('GET', f'/leaderboards?scope={scope}&period={period}&metric={metric}', token=admin)
passed('all 75 leaderboard combinations execute')
# Audio and timestamp coordinates, without external services.
output = io.BytesIO()
with wave.open(output, 'wb') as wav:
    wav.setnchannels(1); wav.setsampwidth(2); wav.setframerate(8000); wav.writeframes(b'\0' * 16000)
vtt = 'WEBVTT\n\n00:00.000 --> 00:01.000\nҺаумыһығыҙ!\n'
body, headers = multipart({'title': 'Acceptance audio', 'license': 'Own recording', 'sourceLang': 'ba', 'rightsConfirmed': 'true', 'transcript': vtt}, 'test.wav', output.getvalue())
audio = request('POST', '/admin/audio/import', body, admin, 201, headers)
draft = next(d for d in request('GET', '/admin/literature/drafts', token=admin) if d['title'] == 'Acceptance audio')
request('PUT', f"/admin/literature/fragments/{draft['id']}/translations", {'languagePair': 'ba-ru', 'correct': 'Здравствуйте!', 'incorrect': 'До свидания!'}, admin)
request('POST', f"/admin/literature/fragments/{draft['id']}/publish", {'languagePair': 'ba-ru'}, admin)
assert len(request('GET', f"/materials/{audio['materialId']}/audio", token=users[1]['accessToken'], expected=206, headers={'Range': 'bytes=0-43'})) == 44
coords = request('GET', f"/fragments/{draft['id']}", token=admin)['source']
assert coords['startTimeMs'] == 0 and coords['endTimeMs'] == 1000
passed('audio upload, timed segmentation and authenticated range playback')
request('GET', '/tasks/daily?langPair=ba-en', token=users[1]['accessToken'], expected=404)
passed('exhausted corpus does not grant repeat XP')
print(f'{len(checks)} acceptance scenarios passed')
