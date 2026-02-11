-- +goose Up
CREATE TABLE items (
    type_id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    volume REAL
);

CREATE VIRTUAL TABLE items_fts USING fts5(
    type_id UNINDEXED,
    name,
    description,
    content=items,
    content_rowid=type_id
);

-- Insert test data (real EVE items)
INSERT INTO items VALUES
    (34, 'Tritanium', 'A heavy, silver-gray metal. Tritanium is the primary building material for most structures and ships in New Eden.', 0.01),
    (35, 'Pyerite', 'A fairly common ore that is very similar to Mexallon in its chemical composition and properties.', 0.01),
    (36, 'Mexallon', 'Malleable precious metal with a high melting point and excellent corrosion resistance.', 0.01),
    (37, 'Isogen', 'Uniquely colored silvery metal. Isogen is considered one of the most important minerals in the universe.', 0.01),
    (38, 'Nocxium', 'A very rare mineral that possesses unique physical and chemical properties.', 0.01),
    (39, 'Zydrine', 'Highly valued ore, with distinctive greenish hue. Zydrine is second only to Megacyte in rarity.', 0.01),
    (40, 'Megacyte', 'The rarest of ores. Megacyte is used extensively in the construction of capital ships.', 0.01),
    (587, 'Rifter', 'The Rifter is a very powerful combat frigate and can easily tackle the best frigates out there.', 24850.0);

-- Populate FTS
INSERT INTO items_fts(type_id, name, description)
SELECT type_id, name, description FROM items;

-- Triggers to keep FTS in sync
CREATE TRIGGER items_ai AFTER INSERT ON items BEGIN
    INSERT INTO items_fts(type_id, name, description)
    VALUES (new.type_id, new.name, new.description);
END;

CREATE TRIGGER items_ad AFTER DELETE ON items BEGIN
    DELETE FROM items_fts WHERE type_id = old.type_id;
END;

CREATE TRIGGER items_au AFTER UPDATE ON items BEGIN
    UPDATE items_fts SET name = new.name, description = new.description
    WHERE type_id = old.type_id;
END;

-- +goose Down
DROP TRIGGER items_au;
DROP TRIGGER items_ad;
DROP TRIGGER items_ai;
DROP TABLE items_fts;
DROP TABLE items;
