-- Login com Google, ultimo acesso e uso da IA (painel do dono).

-- "sub" e o id estavel da conta Google; e-mail pode mudar, sub nao
ALTER TABLE usuarios ADD COLUMN google_sub VARCHAR(64) NULL AFTER senha_hash, ADD UNIQUE KEY uk_usuarios_google (google_sub);

-- atualizado no maximo 1 vez por hora por usuario: o painel mostra quem esta usando
ALTER TABLE usuarios ADD COLUMN ultimo_acesso DATETIME NULL;

-- uso da IA por dia e usuario: a cota gratuita e uma so para o site inteiro
CREATE TABLE IF NOT EXISTS uso_ia (
    dia        DATE            NOT NULL,
    usuario_id BIGINT UNSIGNED NOT NULL,
    perguntas  INT             NOT NULL DEFAULT 0,
    tokens     INT             NOT NULL DEFAULT 0,

    PRIMARY KEY (dia, usuario_id),
    CONSTRAINT fk_uso_ia_usuario FOREIGN KEY (usuario_id) REFERENCES usuarios(id) ON DELETE CASCADE
) ENGINE=InnoDB;
