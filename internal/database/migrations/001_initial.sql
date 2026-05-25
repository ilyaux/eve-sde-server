-- +goose Up
CREATE TABLE categories (
    category_id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    published BOOLEAN NOT NULL DEFAULT 1
);

CREATE TABLE groups (
    group_id INTEGER PRIMARY KEY,
    category_id INTEGER,
    name TEXT NOT NULL,
    published BOOLEAN NOT NULL DEFAULT 1,
    FOREIGN KEY (category_id) REFERENCES categories(category_id)
);

CREATE TABLE items (
    type_id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    volume REAL,
    group_id INTEGER,
    category_id INTEGER,
    published BOOLEAN NOT NULL DEFAULT 1,
    FOREIGN KEY (group_id) REFERENCES groups(group_id),
    FOREIGN KEY (category_id) REFERENCES categories(category_id)
);

CREATE VIRTUAL TABLE items_fts USING fts5(
    type_id UNINDEXED,
    name,
    description,
    content=items,
    content_rowid=type_id
);

CREATE INDEX idx_groups_category_id ON groups(category_id);
CREATE INDEX idx_items_group_id ON items(group_id);
CREATE INDEX idx_items_category_id ON items(category_id);

-- Insert test data (real EVE items)
INSERT INTO categories (category_id, name, published) VALUES
    (4, 'Material', 1),
    (6, 'Ship', 1);

INSERT INTO groups (group_id, category_id, name, published) VALUES
    (18, 4, 'Mineral', 1),
    (25, 6, 'Frigate', 1);

INSERT INTO items (type_id, name, description, volume, group_id, category_id, published) VALUES
    (34, 'Tritanium', 'A heavy, silver-gray metal. Tritanium is the primary building material for most structures and ships in New Eden.', 0.01, 18, 4, 1),
    (35, 'Pyerite', 'A fairly common ore that is very similar to Mexallon in its chemical composition and properties.', 0.01, 18, 4, 1),
    (36, 'Mexallon', 'Malleable precious metal with a high melting point and excellent corrosion resistance.', 0.01, 18, 4, 1),
    (37, 'Isogen', 'Uniquely colored silvery metal. Isogen is considered one of the most important minerals in the universe.', 0.01, 18, 4, 1),
    (38, 'Nocxium', 'A very rare mineral that possesses unique physical and chemical properties.', 0.01, 18, 4, 1),
    (39, 'Zydrine', 'Highly valued ore, with distinctive greenish hue. Zydrine is second only to Megacyte in rarity.', 0.01, 18, 4, 1),
    (40, 'Megacyte', 'The rarest of ores. Megacyte is used extensively in the construction of capital ships.', 0.01, 18, 4, 1),
    (587, 'Rifter', 'The Rifter is a very powerful combat frigate and can easily tackle the best frigates out there.', 24850.0, 25, 6, 1);

-- Populate FTS
INSERT INTO items_fts(rowid, type_id, name, description)
SELECT type_id, type_id, name, description FROM items;

-- Triggers to keep FTS in sync
CREATE TRIGGER items_ai AFTER INSERT ON items BEGIN
    INSERT INTO items_fts(rowid, type_id, name, description)
    VALUES (new.type_id, new.type_id, new.name, new.description);
END;

CREATE TRIGGER items_ad AFTER DELETE ON items BEGIN
    INSERT INTO items_fts(items_fts, rowid, type_id, name, description)
    VALUES ('delete', old.type_id, old.type_id, old.name, old.description);
END;

CREATE TRIGGER items_au AFTER UPDATE ON items BEGIN
    INSERT INTO items_fts(items_fts, rowid, type_id, name, description)
    VALUES ('delete', old.type_id, old.type_id, old.name, old.description);
    INSERT INTO items_fts(rowid, type_id, name, description)
    VALUES (new.type_id, new.type_id, new.name, new.description);
END;

-- +goose Down
DROP TRIGGER items_au;
DROP TRIGGER items_ad;
DROP TRIGGER items_ai;
DROP INDEX idx_items_category_id;
DROP INDEX idx_items_group_id;
DROP INDEX idx_groups_category_id;
DROP TABLE items_fts;
DROP TABLE items;
DROP TABLE groups;
DROP TABLE categories;
