db = db.getSiblingDB('sim_scenario');

db.createCollection('annotations', {
    validator: {
        $jsonSchema: {
            bsonType: 'object',
            required: ['scene_id', 'frame_index', 'objects'],
            properties: {
                scene_id: {
                    bsonType: 'string'
                },
                frame_index: {
                    bsonType: 'int'
                },
                objects: {
                    bsonType: 'array',
                    items: {
                        bsonType: 'object',
                        required: ['track_id', 'type', 'bbox'],
                        properties: {
                            track_id: {
                                bsonType: 'string'
                            },
                            type: {
                                bsonType: 'string'
                            },
                            bbox: {
                                bsonType: 'object',
                                required: ['x', 'y', 'width', 'height'],
                                properties: {
                                    x: { bsonType: 'double' },
                                    y: { bsonType: 'double' },
                                    width: { bsonType: 'double' },
                                    height: { bsonType: 'double' }
                                }
                            },
                            attributes: {
                                bsonType: 'object'
                            }
                        }
                    }
                },
                created_at: {
                    bsonType: 'date'
                },
                updated_at: {
                    bsonType: 'date'
                }
            }
        }
    }
});

db.annotations.createIndex({ scene_id: 1 });
db.annotations.createIndex({ frame_index: 1 });
db.annotations.createIndex({ track_id: 1 });
db.annotations.createIndex({ scene_id: 1, frame_index: 1 });

db.annotations.insertMany([
    {
        scene_id: 'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
        frame_index: 0,
        objects: [
            {
                track_id: 'veh_001',
                type: 'vehicle',
                bbox: { x: 120.5, y: 200.3, width: 80.0, height: 60.0 },
                attributes: { color: 'white', speed: 110.5, lane: 'left' }
            },
            {
                track_id: 'veh_002',
                type: 'vehicle',
                bbox: { x: 400.2, y: 180.1, width: 75.0, height: 55.0 },
                attributes: { color: 'blue', speed: 95.0, lane: 'center' }
            }
        ],
        created_at: new Date(),
        updated_at: new Date()
    },
    {
        scene_id: 'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
        frame_index: 1,
        objects: [
            {
                track_id: 'veh_001',
                type: 'vehicle',
                bbox: { x: 125.0, y: 195.0, width: 80.0, height: 60.0 },
                attributes: { color: 'white', speed: 112.0, lane: 'left' }
            },
            {
                track_id: 'veh_002',
                type: 'vehicle',
                bbox: { x: 395.0, y: 185.0, width: 75.0, height: 55.0 },
                attributes: { color: 'blue', speed: 93.0, lane: 'center' }
            },
            {
                track_id: 'veh_003',
                type: 'vehicle',
                bbox: { x: 50.0, y: 210.0, width: 70.0, height: 50.0 },
                attributes: { color: 'red', speed: 85.0, lane: 'right' }
            }
        ],
        created_at: new Date(),
        updated_at: new Date()
    },
    {
        scene_id: 'b2c3d4e5-f6a7-8901-bcde-f12345678901',
        frame_index: 0,
        objects: [
            {
                track_id: 'ped_001',
                type: 'pedestrian',
                bbox: { x: 300.0, y: 350.0, width: 30.0, height: 60.0 },
                attributes: { action: 'walking', direction: 'left_to_right' }
            },
            {
                track_id: 'veh_010',
                type: 'vehicle',
                bbox: { x: 200.0, y: 250.0, width: 80.0, height: 55.0 },
                attributes: { color: 'silver', speed: 25.0, turn_signal: 'left' }
            }
        ],
        created_at: new Date(),
        updated_at: new Date()
    },
    {
        scene_id: 'b2c3d4e5-f6a7-8901-bcde-f12345678901',
        frame_index: 1,
        objects: [
            {
                track_id: 'ped_001',
                type: 'pedestrian',
                bbox: { x: 310.0, y: 350.0, width: 30.0, height: 60.0 },
                attributes: { action: 'walking', direction: 'left_to_right' }
            },
            {
                track_id: 'veh_010',
                type: 'vehicle',
                bbox: { x: 195.0, y: 248.0, width: 80.0, height: 55.0 },
                attributes: { color: 'silver', speed: 20.0, turn_signal: 'left' }
            },
            {
                track_id: 'ped_002',
                type: 'pedestrian',
                bbox: { x: 450.0, y: 360.0, width: 28.0, height: 58.0 },
                attributes: { action: 'standing', direction: 'stationary' }
            }
        ],
        created_at: new Date(),
        updated_at: new Date()
    },
    {
        scene_id: 'c3d4e5f6-a7b8-9012-cdef-123456789012',
        frame_index: 0,
        objects: [
            {
                track_id: 'obs_001',
                type: 'obstacle',
                bbox: { x: 350.0, y: 300.0, width: 40.0, height: 40.0 },
                attributes: { obstacle_type: 'parked_car', is_dynamic: false }
            },
            {
                track_id: 'veh_020',
                type: 'vehicle',
                bbox: { x: 150.0, y: 280.0, width: 70.0, height: 50.0 },
                attributes: { color: 'black', speed: 8.0, gear: 'D' }
            }
        ],
        created_at: new Date(),
        updated_at: new Date()
    },
    {
        scene_id: 'd4e5f6a7-b8c9-0123-defa-234567890123',
        frame_index: 0,
        objects: [
            {
                track_id: 'veh_030',
                type: 'vehicle',
                bbox: { x: 180.0, y: 220.0, width: 85.0, height: 60.0 },
                attributes: { color: 'gray', speed: 130.0, brake_light: false }
            }
        ],
        created_at: new Date(),
        updated_at: new Date()
    },
    {
        scene_id: 'd4e5f6a7-b8c9-0123-defa-234567890123',
        frame_index: 5,
        objects: [
            {
                track_id: 'veh_030',
                type: 'vehicle',
                bbox: { x: 180.0, y: 220.0, width: 85.0, height: 60.0 },
                attributes: { color: 'gray', speed: 80.0, brake_light: true }
            }
        ],
        created_at: new Date(),
        updated_at: new Date()
    },
    {
        scene_id: 'e5f6a7b8-c9d0-1234-efab-345678901234',
        frame_index: 0,
        objects: [
            {
                track_id: 'ped_010',
                type: 'pedestrian',
                bbox: { x: 420.0, y: 380.0, width: 25.0, height: 55.0 },
                attributes: { action: 'crossing', direction: 'right_to_left', visibility: 'low' }
            }
        ],
        created_at: new Date(),
        updated_at: new Date()
    }
]);
