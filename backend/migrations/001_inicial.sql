-- Jarvis Financeiro - schema inicial (MySQL 8.4)
-- Rodar: mysql -u root -p --port=3307 jarvis < backend/migrations/001_inicial.sql
--
-- Dinheiro em centavos, num BIGINT. Nunca FLOAT: ponto flutuante nao representa
-- 0,10 exatamente e o erro se acumula na soma.

CREATE TABLE IF NOT EXISTS categorias (
    id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    nome               VARCHAR(80)     NOT NULL,
    orcamento_centavos BIGINT          NULL,
    cor                CHAR(7)         NULL,
    criado_em          TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    UNIQUE KEY uk_categorias_nome (nome)
) ENGINE=InnoDB;


CREATE TABLE IF NOT EXISTS transacoes (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    data           DATE            NOT NULL,
    descricao      VARCHAR(255)    NOT NULL,
    valor_centavos BIGINT          NOT NULL,
    tipo           ENUM('debito','credito') NOT NULL,
    categoria_id   BIGINT UNSIGNED NULL,
    conta          VARCHAR(80)     NULL,
    pessoa         VARCHAR(80)     NULL,
    fonte          VARCHAR(20)     NOT NULL,
    hash_externo   CHAR(64)        NOT NULL,
    criado_em      TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id),

    -- a deduplicacao inteira depende deste unique: reimportar o mesmo CSV
    -- vira INSERT IGNORE em vez de linha duplicada
    UNIQUE KEY uk_transacoes_hash (hash_externo),

    KEY idx_transacoes_data (data),
    KEY idx_transacoes_categoria (categoria_id),

    CONSTRAINT fk_transacoes_categoria
        FOREIGN KEY (categoria_id) REFERENCES categorias(id)
        ON DELETE SET NULL
) ENGINE=InnoDB;


-- categorizacao automatica: descricao casando com o padrao recebe a categoria
CREATE TABLE IF NOT EXISTS regras_categoria (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    padrao       VARCHAR(120)    NOT NULL,
    categoria_id BIGINT UNSIGNED NOT NULL,
    prioridade   INT             NOT NULL DEFAULT 0,

    PRIMARY KEY (id),
    KEY idx_regras_prioridade (prioridade DESC),

    CONSTRAINT fk_regras_categoria
        FOREIGN KEY (categoria_id) REFERENCES categorias(id)
        ON DELETE CASCADE
) ENGINE=InnoDB;


-- historico do chat (Fase 4)
CREATE TABLE IF NOT EXISTS conversas (
    id        BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    papel     ENUM('user','assistant') NOT NULL,
    conteudo  TEXT            NOT NULL,
    criado_em TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    KEY idx_conversas_criado_em (criado_em)
) ENGINE=InnoDB;


-- registro de cada importacao, para auditoria e para o front mostrar o historico
CREATE TABLE IF NOT EXISTS importacoes (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    arquivo       VARCHAR(255)    NOT NULL,
    fonte         VARCHAR(20)     NOT NULL,
    conta         VARCHAR(80)     NULL,
    total_linhas  INT             NOT NULL DEFAULT 0,
    importadas    INT             NOT NULL DEFAULT 0,
    duplicadas    INT             NOT NULL DEFAULT 0,
    com_erro      INT             NOT NULL DEFAULT 0,
    criado_em     TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    KEY idx_importacoes_criado_em (criado_em)
) ENGINE=InnoDB;


INSERT IGNORE INTO categorias (nome, cor) VALUES
    ('Alimentação',  '#e0803a'),
    ('Mercado',      '#38e07b'),
    ('Transporte',   '#3aa0e0'),
    ('Moradia',      '#e0c93a'),
    ('Saúde',        '#e05a6a'),
    ('Educação',     '#3ae0d0'),
    ('Lazer',        '#9a7ae0'),
    ('Assinaturas',  '#e03ab0'),
    ('Entrada',      '#38e07b'),
    ('Outros',       '#6f8279');
