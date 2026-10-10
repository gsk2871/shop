CREATE TABLE currency (
 id bigint PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
 code text NOT NULL UNIQUE,
 minor_unit integer NOT NULL CHECK(minor_unit BETWEEN 0 AND 6),
 symbol text NOT NULL, enabled boolean NOT NULL DEFAULT true,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
COMMENT ON TABLE currency IS '币种字典';
COMMENT ON COLUMN currency.minor_unit IS '最小货币单位精度';
