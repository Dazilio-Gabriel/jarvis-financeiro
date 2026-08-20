-- Jarvis Financeiro — schema inicial (MySQL 8.4)
--
-- Rodar:  mysql -u root -p < backend/migrations/001_inicial.sql
--
-- REGRA NÚMERO UM: dinheiro em centavos, num inteiro. Nunca FLOAT, nunca DOUBLE.
-- Ponto flutuante não representa 0,10 exatamente, e o erro se acumula na soma.
-- Se algum dia precisar de fração, use DECIMAL — mas centavos em BIGINT resolve.

CREATE DATABASE IF NOT EXISTS jarvis
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_0900_ai_ci;

USE jarvis;

-- Usuário da aplicação. Nunca conecte a aplicação como root.
-- Troque a senha aqui E no .env (variável MYSQL_DSN).
CREATE USER IF NOT EXISTS 'jarvis'@'localhost' IDENTIFIED BY 'TROQUE_ESTA_SENHA';
GRANT SELECT, INSERT, UPDATE, DELETE ON jarvis.* TO 'jarvis'@'localhost';
FLUSH PRIVILEGES;


CREATE TABLE IF NOT EXISTS categorias (
    id                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    nome               VARCHAR(80)     NOT NULL,
    orcamento_centavos BIGINT          NULL,       -- meta mensal, opcional
    cor                CHAR(7)         NULL,       -- '#RRGGBB'
    criado_em          TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    UNIQUE KEY uk_categorias_nome (nome)
) ENGINE=InnoDB;


CREATE TABLE IF NOT EXISTS transacoes (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    data           DATE            NOT NULL,
    descricao      VARCHAR(255)    NOT NULL,
    valor_centavos BIGINT          NOT NULL,   -- em centavos, sempre
    tipo           ENUM('debito','credito') NOT NULL,
    categoria_id   BIGINT UNSIGNED NULL,       -- NULL = ainda não categorizada
    conta          VARCHAR(80)     NULL,       -- 'Nubank Cartão', 'Nubank Conta'
    pessoa         VARCHAR(80)     NULL,       -- quem da família
    fonte          VARCHAR(20)     NOT NULL,   -- 'csv' | 'pluggy' | 'manual'
    hash_externo   CHAR(64)        NOT NULL,   -- sha256 de data+descricao+valor
    criado_em      TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id),

    -- A deduplicação inteira depende deste índice único.
    -- Reimportar o mesmo CSV vira um INSERT ... ON DUPLICATE KEY UPDATE
    -- (ou INSERT IGNORE) em vez de linha duplicada.
    UNIQUE KEY uk_transacoes_hash (hash_externo),

    KEY idx_transacoes_data (data),
    KEY idx_transacoes_categoria (categoria_id),

    CONSTRAINT fk_transacoes_categoria
        FOREIGN KEY (categoria_id) REFERENCES categorias(id)
        ON DELETE SET NULL
) ENGINE=InnoDB;


-- Categorização automática: se a descrição casar com o padrão, aplica a categoria.
-- 'prioridade' desempata quando mais de uma regra casa (maior vence).
CREATE TABLE IF NOT EXISTS regras_categoria (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    padrao       VARCHAR(120)    NOT NULL,   -- 'IFOOD', 'POSTO%'
    categoria_id BIGINT UNSIGNED NOT NULL,
    prioridade   INT             NOT NULL DEFAULT 0,

    PRIMARY KEY (id),
    KEY idx_regras_prioridade (prioridade DESC),

    CONSTRAINT fk_regras_categoria
        FOREIGN KEY (categoria_id) REFERENCES categorias(id)
        ON DELETE CASCADE
) ENGINE=InnoDB;


-- Histórico do chat com a Claude (Fase 4).
CREATE TABLE IF NOT EXISTS conversas (
    id        BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    papel     ENUM('user','assistant') NOT NULL,
    conteudo  TEXT            NOT NULL,
    criado_em TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    KEY idx_conversas_criado_em (criado_em)
) ENGINE=InnoDB;


-- Categorias iniciais, para não começar com a tabela vazia.
INSERT IGNORE INTO categorias (nome, cor) VALUES
    ('Alimentação',  '#e07a5f'),
    ('Mercado',      '#3d5a80'),
    ('Transporte',   '#81b29a'),
    ('Moradia',      '#f2cc8f'),
    ('Saúde',        '#e63946'),
    ('Educação',     '#457b9d'),
    ('Lazer',        '#9d4edd'),
    ('Assinaturas',  '#2a9d8f'),
    ('Outros',       '#8d99ae');
