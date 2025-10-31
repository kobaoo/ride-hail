begin;

-- Ensure extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- --- ROLES ---
insert into roles (value)
values
    ('PASSENGER'),
    ('DRIVER'),
    ('ADMIN')
on conflict do nothing;

-- --- USER STATUS ---
insert into user_status (value)
values
    ('ACTIVE'),
    ('INACTIVE'),
    ('BANNED')
on conflict do nothing;

-- --- VEHICLE TYPES ---
insert into vehicle_type (value)
values
    ('ECONOMY'),
    ('PREMIUM'),
    ('XL')
on conflict do nothing;

-- --- RIDE STATUS ---
insert into ride_status (value)
values
    ('REQUESTED'),
    ('MATCHED'),
    ('EN_ROUTE'),
    ('ARRIVED'),
    ('IN_PROGRESS'),
    ('COMPLETED'),
    ('CANCELLED')
on conflict do nothing;

-- --- RIDE EVENT TYPES ---
insert into ride_event_type (value)
values
    ('RIDE_REQUESTED'),
    ('DRIVER_MATCHED'),
    ('DRIVER_ARRIVED'),
    ('RIDE_STARTED'),
    ('RIDE_COMPLETED'),
    ('RIDE_CANCELLED'),
    ('STATUS_CHANGED'),
    ('LOCATION_UPDATED'),
    ('FARE_ADJUSTED')
on conflict do nothing;

-- --- USERS ---
insert into users (id, email, role, status, password_hash, attrs)
values
    -- Passengers
    ('550e8400-e29b-41d4-a716-446655440001', 'passenger1@example.com', 'PASSENGER', 'ACTIVE', 'hashed_passenger1', '{"name":"Alice Passenger"}'),
    ('550e8400-e29b-41d4-a716-446655440002', 'passenger2@example.com', 'PASSENGER', 'ACTIVE', 'hashed_passenger2', '{"name":"Bob Passenger"}'),

    -- Drivers
    ('660e8400-e29b-41d4-a716-446655440001', 'driver1@example.com', 'DRIVER', 'ACTIVE', 'hashed_driver1', '{"name":"Charlie Driver"}'),
    ('660e8400-e29b-41d4-a716-446655440002', 'driver2@example.com', 'DRIVER', 'ACTIVE', 'hashed_driver2', '{"name":"Dana Driver"}')
on conflict do nothing;

-- --- COORDINATES ---
insert into coordinates (id, entity_id, entity_type, address, latitude, longitude, fare_amount, distance_km, duration_minutes, is_current)
values
    -- Passenger locations
    ('770e8400-e29b-41d4-a716-446655440001', '550e8400-e29b-41d4-a716-446655440001', 'passenger', 'Almaty Center, Abay Ave 10', 43.238949, 76.889709, null, null, null, true),
    ('770e8400-e29b-41d4-a716-446655440002', '550e8400-e29b-41d4-a716-446655440002', 'passenger', 'Almaty Towers, Dostyk Ave 42', 43.236389, 76.945556, null, null, null, true),

    -- Driver locations
    ('880e8400-e29b-41d4-a716-446655440001', '660e8400-e29b-41d4-a716-446655440001', 'driver', 'Bogenbai Batyr St 150', 43.250000, 76.920000, null, null, null, true),
    ('880e8400-e29b-41d4-a716-446655440002', '660e8400-e29b-41d4-a716-446655440002', 'driver', 'Tashkentskaya St 25', 43.240000, 76.880000, null, null, null, true)
on conflict do nothing;

-- --- RIDES ---
insert into rides (
    id,
    ride_number,
    passenger_id,
    driver_id,
    vehicle_type,
    status,
    priority,
    requested_at,
    matched_at,
    started_at,
    completed_at,
    estimated_fare,
    final_fare,
    pickup_coordinate_id,
    destination_coordinate_id
) values
    -- Completed ride
    ('990e8400-e29b-41d4-a716-446655440001',
     'RIDE_20241010_001',
     '550e8400-e29b-41d4-a716-446655440001',
     '660e8400-e29b-41d4-a716-446655440001',
     'ECONOMY',
     'COMPLETED',
     1,
     now() - interval '40 minutes',
     now() - interval '35 minutes',
     now() - interval '20 minutes',
     now() - interval '5 minutes',
     1300.00,
     1300.00,
     '770e8400-e29b-41d4-a716-446655440001',
     '770e8400-e29b-41d4-a716-446655440002'),

    -- Requested ride waiting for match
    ('990e8400-e29b-41d4-a716-446655440002',
     'RIDE_20241010_002',
     '550e8400-e29b-41d4-a716-446655440002',
     null,
     'PREMIUM',
     'REQUESTED',
     1,
     now() - interval '5 minutes',
     null,
     null,
     null,
     2200.00,
     null,
     '770e8400-e29b-41d4-a716-446655440002',
     '770e8400-e29b-41d4-a716-446655440001')
on conflict do nothing;

-- --- RIDE EVENTS ---
insert into ride_events (ride_id, event_type, event_data)
values
    ('990e8400-e29b-41d4-a716-446655440001', 'RIDE_REQUESTED', '{"pickup":"Abay Ave 10","destination":"Dostyk Ave 42"}'),
    ('990e8400-e29b-41d4-a716-446655440001', 'DRIVER_MATCHED', '{"driver_id":"660e8400-e29b-41d4-a716-446655440001"}'),
    ('990e8400-e29b-41d4-a716-446655440001', 'RIDE_COMPLETED', '{"fare":1300,"distance_km":5.4}'),

    ('990e8400-e29b-41d4-a716-446655440002', 'RIDE_REQUESTED', '{"pickup":"Dostyk Ave 42","destination":"Abay Ave 10"}');

commit;
