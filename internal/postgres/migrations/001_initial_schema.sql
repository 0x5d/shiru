CREATE TABLE users (
    id UUID PRIMARY KEY,
    handle TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_settings (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    jlpt_level TEXT NOT NULL DEFAULT 'N5' CHECK (jlpt_level IN ('N5','N4','N3','N2','N1')),
    story_word_target INT NOT NULL DEFAULT 100 CHECK (story_word_target BETWEEN 50 AND 500),
    wanikani_api_key TEXT,
    wanikani_last_synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE vocab_entries (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    surface TEXT NOT NULL,
    normalized_surface TEXT NOT NULL,
    meaning TEXT,
    reading TEXT,
    source TEXT NOT NULL CHECK (source IN ('manual','wanikani')),
    source_ref TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, normalized_surface)
);

CREATE TABLE tags (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, name)
);

CREATE TABLE vocab_entry_tags (
    vocab_entry_id UUID NOT NULL REFERENCES vocab_entries(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    rank SMALLINT NOT NULL CHECK (rank BETWEEN 1 AND 3),
    PRIMARY KEY (vocab_entry_id, tag_id)
);

CREATE TABLE stories (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    topic TEXT NOT NULL,
    tone TEXT NOT NULL CHECK (tone IN ('funny','shocking')),
    jlpt_level TEXT NOT NULL,
    target_word_count INT NOT NULL,
    actual_word_count INT NOT NULL,
    content TEXT NOT NULL,
    used_vocab_count INT NOT NULL,
    source_tag_names TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE story_vocab_entries (
    story_id UUID NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    vocab_entry_id UUID NOT NULL REFERENCES vocab_entries(id) ON DELETE CASCADE,
    PRIMARY KEY (story_id, vocab_entry_id)
);

CREATE TABLE story_audio (
    story_id UUID PRIMARY KEY REFERENCES stories(id) ON DELETE CASCADE,
    voice_id TEXT NOT NULL,
    audio_format TEXT NOT NULL,
    storage_path TEXT NOT NULL,
    duration_ms INT,
    checksum TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE topic_snapshots (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    topics TEXT[] NOT NULL,
    prompt_version TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Seed default user
INSERT INTO users (id, handle) VALUES ('00000000-0000-0000-0000-000000000001', 'default_user');
INSERT INTO user_settings (user_id) VALUES ('00000000-0000-0000-0000-000000000001');
