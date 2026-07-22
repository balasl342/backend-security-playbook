CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS customers (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT NOT NULL,
    email          TEXT NOT NULL,
    phone          TEXT NOT NULL,

    -- Mode A (plaintext, relies on DB-level TDE): populated when the service
    -- runs with crypto.mode=plaintext. NULL in Mode B.
    ssn            TEXT,
    credit_card    TEXT,
    address        TEXT,

    -- Mode B (application-level AES-256-GCM / envelope encryption):
    -- cipher_text holds a JSON-encoded map of field name -> base64
    -- ciphertext (nonce || ciphertext || tag) for ssn, credit_card, and
    -- address. key_version identifies which master/data key version
    -- encrypted this row, so old rows keep decrypting correctly after a key
    -- rotation. Both are NULL in Mode A.
    cipher_text    BYTEA,
    key_version    INTEGER,

    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT customers_email_unique UNIQUE (email),
    CONSTRAINT customers_mode_consistency CHECK (
        (ssn IS NOT NULL AND credit_card IS NOT NULL AND address IS NOT NULL
            AND cipher_text IS NULL AND key_version IS NULL)
        OR
        (ssn IS NULL AND credit_card IS NULL AND address IS NULL
            AND cipher_text IS NOT NULL AND key_version IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_customers_key_version ON customers (key_version);
