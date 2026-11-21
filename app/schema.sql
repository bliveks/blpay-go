CREATE TABLE IF NOT EXISTS istifadeciler (
 id BIGSERIAL PRIMARY KEY, ad VARCHAR(50) NOT NULL, soyad VARCHAR(50) NOT NULL,
 email VARCHAR(255) NOT NULL UNIQUE, sifre TEXT NOT NULL, yaradilma_tarixi TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 son_giris TIMESTAMPTZ, qeydiyyat_tamamlanib BOOLEAN NOT NULL DEFAULT TRUE,
 status TEXT NOT NULL DEFAULT 'aktiv' CHECK (status IN ('aktiv','deaktiv')),
 rol TEXT NOT NULL DEFAULT 'user' CHECK (rol IN ('admin','user'))
);
CREATE TABLE IF NOT EXISTS kateqoriyalar (
 id BIGSERIAL PRIMARY KEY, istifadeci_id BIGINT REFERENCES istifadeciler(id) ON DELETE CASCADE,
 ad VARCHAR(80) NOT NULL, tip TEXT NOT NULL CHECK (tip IN ('gelir','xerc')), limit_mebleq NUMERIC(12,2),
 standart BOOLEAN NOT NULL DEFAULT FALSE, yaradilma_tarixi TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS gelirler (
 id BIGSERIAL PRIMARY KEY, istifadeci_id BIGINT NOT NULL REFERENCES istifadeciler(id) ON DELETE CASCADE,
 kateqoriya_id BIGINT NOT NULL REFERENCES kateqoriyalar(id), mebleq NUMERIC(12,2) NOT NULL CHECK(mebleq > 0),
 tarix DATE NOT NULL, hesab_novu TEXT NOT NULL CHECK(hesab_novu IN ('nagd','kredit_kart','debet_kart')), qeyd TEXT NOT NULL DEFAULT '', yaradilma_tarixi TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS xercler (
 id BIGSERIAL PRIMARY KEY, istifadeci_id BIGINT NOT NULL REFERENCES istifadeciler(id) ON DELETE CASCADE,
 kateqoriya_id BIGINT NOT NULL REFERENCES kateqoriyalar(id), mebleq NUMERIC(12,2) NOT NULL CHECK(mebleq > 0),
 tarix DATE NOT NULL, hesab_novu TEXT NOT NULL CHECK(hesab_novu IN ('nagd','kredit_kart','debet_kart')), qeyd TEXT NOT NULL DEFAULT '', yaradilma_tarixi TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS hedefler (
 id BIGSERIAL PRIMARY KEY, istifadeci_id BIGINT NOT NULL REFERENCES istifadeciler(id) ON DELETE CASCADE,
 ad VARCHAR(120) NOT NULL, hedef_mebleq NUMERIC(12,2) NOT NULL CHECK(hedef_mebleq > 0), cari_mebleq NUMERIC(12,2) NOT NULL DEFAULT 0,
 baslangic_tarixi DATE NOT NULL, hedef_tarixi DATE NOT NULL, prioritet TEXT NOT NULL DEFAULT 'orta' CHECK(prioritet IN ('yuksek','orta','asagi')),
 status TEXT NOT NULL DEFAULT 'aktiv' CHECK(status IN ('aktiv','tamamlanib','legv_edilib')), qeyd TEXT NOT NULL DEFAULT '', yaradilma_tarixi TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS hedef_emeliyyatlar (
 id BIGSERIAL PRIMARY KEY, hedef_id BIGINT NOT NULL REFERENCES hedefler(id) ON DELETE CASCADE,
 istifadeci_id BIGINT NOT NULL REFERENCES istifadeciler(id) ON DELETE CASCADE, mebleq NUMERIC(12,2) NOT NULL,
 emeliyyat_novu TEXT NOT NULL CHECK(emeliyyat_novu IN ('artirma','azaltma')), tarix DATE NOT NULL, qeyd TEXT NOT NULL DEFAULT '', yaradilma_tarixi TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS audit_log (
 id BIGSERIAL PRIMARY KEY, user_id BIGINT REFERENCES istifadeciler(id) ON DELETE SET NULL,
 action VARCHAR(255) NOT NULL, details TEXT NOT NULL DEFAULT '', ip_address VARCHAR(45), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS giris_cehdleri (id BIGSERIAL PRIMARY KEY, ip VARCHAR(45) NOT NULL, tarix TIMESTAMPTZ NOT NULL DEFAULT NOW());
CREATE INDEX IF NOT EXISTS gelirler_user_date_idx ON gelirler(istifadeci_id, tarix DESC);
CREATE INDEX IF NOT EXISTS xercler_user_date_idx ON xercler(istifadeci_id, tarix DESC);
CREATE INDEX IF NOT EXISTS kateqoriyalar_user_idx ON kateqoriyalar(istifadeci_id, tip);
CREATE INDEX IF NOT EXISTS hedefler_user_status_date_idx ON hedefler(istifadeci_id, status, hedef_tarixi);
CREATE INDEX IF NOT EXISTS hedef_emeliyyatlar_goal_idx ON hedef_emeliyyatlar(hedef_id, yaradilma_tarixi DESC);
CREATE INDEX IF NOT EXISTS hedef_emeliyyatlar_user_idx ON hedef_emeliyyatlar(istifadeci_id, yaradilma_tarixi DESC);
CREATE INDEX IF NOT EXISTS audit_log_user_date_idx ON audit_log(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS giris_cehdleri_ip_date_idx ON giris_cehdleri(ip, tarix DESC);
CREATE UNIQUE INDEX IF NOT EXISTS standard_category_key_idx
  ON kateqoriyalar(ad, tip) WHERE standart=TRUE AND istifadeci_id IS NULL;
INSERT INTO kateqoriyalar(istifadeci_id,ad,tip,standart)
SELECT NULL, c.ad, c.tip, TRUE FROM (VALUES
 ('Maaş','gelir'),('Biznes','gelir'),('İnvestisiya','gelir'),('Digər','gelir'),
 ('Qida','xerc'),('Kommunal','xerc'),('Nəqliyyat','xerc'),('Təhsil','xerc'),('Səhiyyə','xerc'),('Əyləncə','xerc'),('Geyim','xerc'),('Digər','xerc')
) AS c(ad,tip)
ON CONFLICT (ad,tip) WHERE standart=TRUE AND istifadeci_id IS NULL DO NOTHING;

-- These tables are used only by the Go server and deny direct Supabase Data API access.
ALTER TABLE istifadeciler ENABLE ROW LEVEL SECURITY;
ALTER TABLE kateqoriyalar ENABLE ROW LEVEL SECURITY;
ALTER TABLE gelirler ENABLE ROW LEVEL SECURITY;
ALTER TABLE xercler ENABLE ROW LEVEL SECURITY;
ALTER TABLE hedefler ENABLE ROW LEVEL SECURITY;
ALTER TABLE hedef_emeliyyatlar ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE giris_cehdleri ENABLE ROW LEVEL SECURITY;
