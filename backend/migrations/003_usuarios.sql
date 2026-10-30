-- Usuarios, familias e sessoes. Os dados passam a pertencer a uma familia: quem entra com o
-- codigo de convite ve e edita os mesmos lancamentos (o "familiar" do Jarvis).
-- O que ja existia no banco vai para a familia 1; o primeiro usuario cadastrado assume ela.

CREATE TABLE IF NOT EXISTS familias (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    nome           VARCHAR(80)     NOT NULL,
    codigo_convite CHAR(10)        NOT NULL,
    criado_em      TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    UNIQUE KEY uk_familias_convite (codigo_convite)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS usuarios (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    familia_id  BIGINT UNSIGNED NOT NULL,
    nome        VARCHAR(80)     NOT NULL,
    email       VARCHAR(190)    NOT NULL,
    senha_hash  VARCHAR(100)    NOT NULL, -- bcrypt; a senha em si nunca e gravada
    criado_em   TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    UNIQUE KEY uk_usuarios_email (email),
    CONSTRAINT fk_usuarios_familia FOREIGN KEY (familia_id) REFERENCES familias(id)
) ENGINE=InnoDB;

-- guarda so o sha256 do token: quem ler o banco nao consegue usar a sessao de ninguem
CREATE TABLE IF NOT EXISTS sessoes (
    token_hash  CHAR(64)        NOT NULL,
    usuario_id  BIGINT UNSIGNED NOT NULL,
    expira_em   DATETIME        NOT NULL,
    criado_em   TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (token_hash),
    KEY idx_sessoes_usuario (usuario_id),
    CONSTRAINT fk_sessoes_usuario FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE
) ENGINE=InnoDB;

INSERT IGNORE INTO familias (id, nome, codigo_convite)
VALUES (1, 'Minha família', UPPER(SUBSTRING(MD5(RAND()), 1, 10)));

-- familia_id em tudo que e dado da familia; DEFAULT 1 preenche o que ja existe
ALTER TABLE categorias       ADD COLUMN familia_id BIGINT UNSIGNED NOT NULL DEFAULT 1 AFTER id;
ALTER TABLE regras_categoria ADD COLUMN familia_id BIGINT UNSIGNED NOT NULL DEFAULT 1 AFTER id;
ALTER TABLE transacoes       ADD COLUMN familia_id BIGINT UNSIGNED NOT NULL DEFAULT 1 AFTER id;
ALTER TABLE importacoes      ADD COLUMN familia_id BIGINT UNSIGNED NOT NULL DEFAULT 1 AFTER id;

ALTER TABLE categorias       ALTER COLUMN familia_id DROP DEFAULT;
ALTER TABLE regras_categoria ALTER COLUMN familia_id DROP DEFAULT;
ALTER TABLE transacoes       ALTER COLUMN familia_id DROP DEFAULT;
ALTER TABLE importacoes      ALTER COLUMN familia_id DROP DEFAULT;

-- nome de categoria e hash de deduplicacao passam a ser unicos POR familia:
-- duas familias podem ter "Mercado" e importar a mesma fatura de exemplo
ALTER TABLE categorias DROP INDEX uk_categorias_nome, ADD UNIQUE KEY uk_categorias_familia_nome (familia_id, nome);
ALTER TABLE transacoes DROP INDEX uk_transacoes_hash, ADD UNIQUE KEY uk_transacoes_familia_hash (familia_id, hash_externo);

ALTER TABLE categorias       ADD CONSTRAINT fk_categorias_familia FOREIGN KEY (familia_id) REFERENCES familias(id);
ALTER TABLE regras_categoria ADD CONSTRAINT fk_regras_familia     FOREIGN KEY (familia_id) REFERENCES familias(id);
ALTER TABLE transacoes       ADD CONSTRAINT fk_transacoes_familia FOREIGN KEY (familia_id) REFERENCES familias(id);
ALTER TABLE importacoes      ADD CONSTRAINT fk_importacoes_familia FOREIGN KEY (familia_id) REFERENCES familias(id);

CREATE INDEX idx_transacoes_familia_mes ON transacoes (familia_id, mes_ref);
