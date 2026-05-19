-- Realistic demo data for users + subscriptions + audit_logs.
--
-- All UUIDs are hard-coded so re-running is a no-op (ON CONFLICT
-- DO NOTHING). Password hash is for "demo1234" (bcrypt cost 10).
--
-- Email layout: 30 mixed Russian/Belarusian developer-style accounts
-- spanning the year. Roles: 28 user, 1 admin, 1 moderator.

INSERT INTO users (id, email, username, password_hash, first_name, last_name, role, status, provider, email_verified, created_at, last_login_at)
VALUES
  ('11111111-1111-1111-1111-000000000001','admin@realsync.io','admin','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Анна','Каминская','admin','active','local',true, NOW() - INTERVAL '120 days', NOW() - INTERVAL '1 day'),
  ('11111111-1111-1111-1111-000000000002','dmitry.ivanov@realsync.io','dmitry_ivanov','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Дмитрий','Иванов','user','active','local',true, NOW() - INTERVAL '90 days', NOW() - INTERVAL '2 days'),
  ('11111111-1111-1111-1111-000000000003','olga.petrova@realsync.io','olga_petrova','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Ольга','Петрова','user','active','local',true, NOW() - INTERVAL '85 days', NOW() - INTERVAL '3 days'),
  ('11111111-1111-1111-1111-000000000004','sergey.volkov@realsync.io','sergey_volkov','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Сергей','Волков','user','active','local',true, NOW() - INTERVAL '80 days', NOW() - INTERVAL '5 days'),
  ('11111111-1111-1111-1111-000000000005','anastasia.kovaleva@realsync.io','anastasia_kovaleva','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Анастасия','Ковалёва','user','active','local',true, NOW() - INTERVAL '75 days', NOW() - INTERVAL '1 day'),
  ('11111111-1111-1111-1111-000000000006','maksim.shevchenko@realsync.io','maksim_shevchenko','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Максим','Шевченко','user','active','local',true, NOW() - INTERVAL '70 days', NOW() - INTERVAL '4 days'),
  ('11111111-1111-1111-1111-000000000007','elena.morozova@realsync.io','elena_morozova','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Елена','Морозова','admin','active','local',true, NOW() - INTERVAL '65 days', NOW() - INTERVAL '6 hours'),
  ('11111111-1111-1111-1111-000000000008','vladimir.kuznetsov@realsync.io','vladimir_kuznetsov','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Владимир','Кузнецов','user','active','local',true, NOW() - INTERVAL '60 days', NOW() - INTERVAL '2 days'),
  ('11111111-1111-1111-1111-000000000009','tatyana.lebedeva@realsync.io','tatyana_lebedeva','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Татьяна','Лебедева','user','active','local',true, NOW() - INTERVAL '55 days', NOW() - INTERVAL '12 hours'),
  ('11111111-1111-1111-1111-000000000010','andrey.bogdanov@realsync.io','andrey_bogdanov','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Андрей','Богданов','user','active','local',true, NOW() - INTERVAL '50 days', NOW() - INTERVAL '1 day'),
  ('11111111-1111-1111-1111-000000000011','natalia.fedorova@realsync.io','natalia_fedorova','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Наталья','Фёдорова','user','active','local',true, NOW() - INTERVAL '45 days', NOW() - INTERVAL '3 days'),
  ('11111111-1111-1111-1111-000000000012','pavel.sokolov@realsync.io','pavel_sokolov','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Павел','Соколов','user','active','local',true, NOW() - INTERVAL '40 days', NOW() - INTERVAL '7 days'),
  ('11111111-1111-1111-1111-000000000013','irina.smirnova@realsync.io','irina_smirnova','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Ирина','Смирнова','user','active','local',true, NOW() - INTERVAL '38 days', NOW() - INTERVAL '4 days'),
  ('11111111-1111-1111-1111-000000000014','aleksey.gusev@realsync.io','aleksey_gusev','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Алексей','Гусев','user','active','local',true, NOW() - INTERVAL '35 days', NOW() - INTERVAL '2 days'),
  ('11111111-1111-1111-1111-000000000015','marina.zaytseva@realsync.io','marina_zaytseva','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Марина','Зайцева','user','active','local',true, NOW() - INTERVAL '32 days', NOW() - INTERVAL '8 hours'),
  ('11111111-1111-1111-1111-000000000016','denis.romanov@realsync.io','denis_romanov','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Денис','Романов','user','active','local',true, NOW() - INTERVAL '30 days', NOW() - INTERVAL '5 days'),
  ('11111111-1111-1111-1111-000000000017','svetlana.belova@realsync.io','svetlana_belova','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Светлана','Белова','user','active','local',true, NOW() - INTERVAL '28 days', NOW() - INTERVAL '1 day'),
  ('11111111-1111-1111-1111-000000000018','igor.medvedev@realsync.io','igor_medvedev','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Игорь','Медведев','user','active','local',true, NOW() - INTERVAL '25 days', NOW() - INTERVAL '6 days'),
  ('11111111-1111-1111-1111-000000000019','yulia.komarova@realsync.io','yulia_komarova','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Юлия','Комарова','user','active','local',true, NOW() - INTERVAL '22 days', NOW() - INTERVAL '3 days'),
  ('11111111-1111-1111-1111-000000000020','viktor.lobanov@realsync.io','viktor_lobanov','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Виктор','Лобанов','user','active','local',true, NOW() - INTERVAL '20 days', NOW() - INTERVAL '10 hours'),
  ('11111111-1111-1111-1111-000000000021','alina.zhukova@realsync.io','alina_zhukova','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Алина','Жукова','user','active','local',true, NOW() - INTERVAL '18 days', NOW() - INTERVAL '1 day'),
  ('11111111-1111-1111-1111-000000000022','roman.kiselev@realsync.io','roman_kiselev','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Роман','Киселёв','user','active','local',true, NOW() - INTERVAL '15 days', NOW() - INTERVAL '2 hours'),
  ('11111111-1111-1111-1111-000000000023','daria.ershova@realsync.io','daria_ershova','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Дарья','Ершова','user','active','local',true, NOW() - INTERVAL '12 days', NOW() - INTERVAL '5 hours'),
  ('11111111-1111-1111-1111-000000000024','nikita.tsvetkov@realsync.io','nikita_tsvetkov','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Никита','Цветков','user','active','local',true, NOW() - INTERVAL '10 days', NOW() - INTERVAL '1 day'),
  ('11111111-1111-1111-1111-000000000025','vera.lazareva@realsync.io','vera_lazareva','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Вера','Лазарева','user','suspended','local',true, NOW() - INTERVAL '8 days', NOW() - INTERVAL '4 days'),
  ('11111111-1111-1111-1111-000000000026','egor.smirnov@realsync.io','egor_smirnov','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Егор','Смирнов','user','active','local',true, NOW() - INTERVAL '6 days', NOW() - INTERVAL '12 hours'),
  ('11111111-1111-1111-1111-000000000027','kristina.popova@realsync.io','kristina_popova','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Кристина','Попова','user','active','local',true, NOW() - INTERVAL '5 days', NOW() - INTERVAL '6 hours'),
  ('11111111-1111-1111-1111-000000000028','stanislav.orlov@realsync.io','stanislav_orlov','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Станислав','Орлов','user','active','local',true, NOW() - INTERVAL '4 days', NOW() - INTERVAL '2 hours'),
  ('11111111-1111-1111-1111-000000000029','polina.golubeva@realsync.io','polina_golubeva','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Полина','Голубева','user','active','local',true, NOW() - INTERVAL '3 days', NOW() - INTERVAL '1 hour'),
  ('11111111-1111-1111-1111-000000000030','timofey.solovyev@realsync.io','timofey_solovyev','$2a$10$Sjc7XzkGm0XJ/dY8rdKKpO5tIyOaoVZBOXkBpY/0V/UJqHkz5p3rW','Тимофей','Соловьёв','user','active','local',true, NOW() - INTERVAL '2 days', NOW() - INTERVAL '30 minutes')
ON CONFLICT (id) DO NOTHING;

-- Subscriptions: mix of starter / pro / team / cancelled.
INSERT INTO subscriptions (id, user_id, tier, status, start_date, end_date, auto_renew, max_users, max_storage_gb, features, metadata, created_at)
VALUES
  ('22222222-1111-0001-0000-000000000001','11111111-1111-1111-1111-000000000002','pro','active', NOW() - INTERVAL '85 days', NOW() + INTERVAL '5 days', true, 1, 5, '{ai_priority,exports,history}', '{"price":65,"currency":"BYN"}', NOW() - INTERVAL '85 days'),
  ('22222222-1111-0001-0000-000000000002','11111111-1111-1111-1111-000000000003','team','active', NOW() - INTERVAL '70 days', NOW() + INTERVAL '20 days', true, 5, 25, '{ai_priority,exports,history,team_dashboards}', '{"price":159,"currency":"BYN"}', NOW() - INTERVAL '70 days'),
  ('22222222-1111-0001-0000-000000000003','11111111-1111-1111-1111-000000000004','starter','active', NOW() - INTERVAL '60 days', NOW() + INTERVAL '30 days', true, 1, 2, '{ai_basic,history}', '{"price":29,"currency":"BYN"}', NOW() - INTERVAL '60 days'),
  ('22222222-1111-0001-0000-000000000004','11111111-1111-1111-1111-000000000010','pro','active', NOW() - INTERVAL '45 days', NOW() + INTERVAL '15 days', true, 1, 5, '{ai_priority,exports,history}', '{"price":65,"currency":"BYN"}', NOW() - INTERVAL '45 days'),
  ('22222222-1111-0001-0000-000000000005','11111111-1111-1111-1111-000000000012','starter','canceled', NOW() - INTERVAL '30 days', NOW() - INTERVAL '2 days', false, 1, 2, '{ai_basic,history}', '{"price":29,"currency":"BYN","cancel_reason":"too_expensive"}', NOW() - INTERVAL '30 days'),
  ('22222222-1111-0001-0000-000000000006','11111111-1111-1111-1111-000000000014','pro','active', NOW() - INTERVAL '25 days', NOW() + INTERVAL '5 days', true, 1, 5, '{ai_priority,exports,history}', '{"price":65,"currency":"BYN"}', NOW() - INTERVAL '25 days'),
  ('22222222-1111-0001-0000-000000000007','11111111-1111-1111-1111-000000000016','team','active', NOW() - INTERVAL '20 days', NOW() + INTERVAL '40 days', true, 5, 25, '{ai_priority,exports,history,team_dashboards}', '{"price":159,"currency":"BYN"}', NOW() - INTERVAL '20 days'),
  ('22222222-1111-0001-0000-000000000008','11111111-1111-1111-1111-000000000020','starter','active', NOW() - INTERVAL '15 days', NOW() + INTERVAL '15 days', true, 1, 2, '{ai_basic,history}', '{"price":29,"currency":"BYN"}', NOW() - INTERVAL '15 days'),
  ('22222222-1111-0001-0000-000000000009','11111111-1111-1111-1111-000000000022','pro','active', NOW() - INTERVAL '10 days', NOW() + INTERVAL '20 days', true, 1, 5, '{ai_priority,exports,history}', '{"price":65,"currency":"BYN"}', NOW() - INTERVAL '10 days'),
  ('22222222-1111-0001-0000-000000000010','11111111-1111-1111-1111-000000000024','starter','active', NOW() - INTERVAL '7 days', NOW() + INTERVAL '23 days', true, 1, 2, '{ai_basic,history}', '{"price":29,"currency":"BYN"}', NOW() - INTERVAL '7 days')
ON CONFLICT (id) DO NOTHING;

-- Audit log entries for admin activity.
INSERT INTO audit_logs (id, admin_id, admin_email, action, resource_type, resource_id, ip_address, user_agent, created_at, details)
VALUES
  ('33333333-1111-0001-0000-000000000001','11111111-1111-1111-1111-000000000001','admin@realsync.io','suspend_user','user','11111111-1111-1111-1111-000000000025','192.168.1.10','Mozilla/5.0', NOW() - INTERVAL '4 days','{"reason":"подозрительная активность"}'),
  ('33333333-1111-0001-0000-000000000002','11111111-1111-1111-1111-000000000001','admin@realsync.io','change_subscription','subscription','22222222-1111-0001-0000-000000000007','192.168.1.10','Mozilla/5.0', NOW() - INTERVAL '20 days','{}'),
  ('33333333-1111-0001-0000-000000000003','11111111-1111-1111-1111-000000000001','admin@realsync.io','change_role','user','11111111-1111-1111-1111-000000000007','192.168.1.10','Mozilla/5.0', NOW() - INTERVAL '65 days','{"from":"user","to":"admin"}'),
  ('33333333-1111-0001-0000-000000000004','11111111-1111-1111-1111-000000000001','admin@realsync.io','change_subscription','subscription','22222222-1111-0001-0000-000000000005','192.168.1.10','Mozilla/5.0', NOW() - INTERVAL '2 days','{}'),
  ('33333333-1111-0001-0000-000000000005','11111111-1111-1111-1111-000000000001','admin@realsync.io','update','system',NULL,'192.168.1.10','Mozilla/5.0', NOW() - INTERVAL '10 days','{"key":"max_interview_duration","old":60,"new":90}')
ON CONFLICT (id) DO NOTHING;
