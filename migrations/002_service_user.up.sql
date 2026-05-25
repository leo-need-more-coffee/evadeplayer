INSERT INTO users (id, email, password)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'service@evadeplayer.local',
    'service-key-auth'
)
ON CONFLICT (id) DO NOTHING;
