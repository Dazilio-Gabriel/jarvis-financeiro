-- Caixinha com objetivo e rendimento.
-- meta_centavos: objetivo na moeda da conta (0 = sem meta).
-- rende_cdi: quanto rende em % do CDI (100 = 100% do CDI; 0 = nao rende ou nao informado).
-- Um ALTER por coluna: o TiDB Cloud nao aceita varias mudancas de schema no mesmo comando
-- em todas as versoes.

ALTER TABLE ativos ADD COLUMN meta_centavos BIGINT NOT NULL DEFAULT 0;

ALTER TABLE ativos ADD COLUMN rende_cdi SMALLINT NOT NULL DEFAULT 0;
