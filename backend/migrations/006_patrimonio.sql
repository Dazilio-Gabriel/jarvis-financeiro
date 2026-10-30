-- Patrimonio: caixinhas, contas em outra moeda e investimentos da familia.
-- Saldo sempre em centavos DA MOEDA da conta; a conversao para real e feita na hora, pela
-- cotacao do dia, e nunca gravada (cotacao muda; o saldo que a pessoa informou, nao).

CREATE TABLE IF NOT EXISTS ativos (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    familia_id     BIGINT UNSIGNED NOT NULL,
    nome           VARCHAR(60)     NOT NULL,
    tipo           VARCHAR(20)     NOT NULL, -- caixinha | conta | investimento | outro
    moeda          CHAR(3)         NOT NULL, -- BRL | USD | EUR
    saldo_centavos BIGINT          NOT NULL DEFAULT 0,
    atualizado_em  DATETIME        NOT NULL,
    criado_em      TIMESTAMP       NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    KEY idx_ativos_familia (familia_id),
    CONSTRAINT fk_ativos_familia FOREIGN KEY (familia_id) REFERENCES familias(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

-- cada atualizacao de saldo vira um ponto: e daqui que sai a evolucao mes a mes
CREATE TABLE IF NOT EXISTS ativos_historico (
    id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    ativo_id       BIGINT UNSIGNED NOT NULL,
    saldo_centavos BIGINT          NOT NULL,
    registrado_em  DATETIME        NOT NULL,

    PRIMARY KEY (id),
    KEY idx_historico_ativo (ativo_id, registrado_em),
    CONSTRAINT fk_historico_ativo FOREIGN KEY (ativo_id) REFERENCES ativos(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
