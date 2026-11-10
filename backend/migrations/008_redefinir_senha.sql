-- "Esqueci minha senha": o token vai no link do e-mail e aqui fica so o sha256 dele.
-- Uso unico: a linha e apagada quando a senha e trocada, e pedir de novo invalida o link anterior.
CREATE TABLE IF NOT EXISTS redefinicoes_senha (
    token_hash  CHAR(64)        NOT NULL,
    usuario_id  BIGINT UNSIGNED NOT NULL,
    expira_em   DATETIME        NOT NULL,
    criado_em   TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (token_hash),
    KEY idx_redefinicoes_usuario (usuario_id),
    CONSTRAINT fk_redefinicoes_usuario FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
