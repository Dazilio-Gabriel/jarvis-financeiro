-- Mes de referencia de cada transacao.
-- Na fatura do cartao, a compra de 20/10 e paga na fatura de novembro: o mes que importa para o
-- orcamento da familia e o da fatura, nao o da compra. Extrato e lancamento manual usam o mes da data.
-- Os totais por mes (dashboard, chat, estatisticas) agrupam por esta coluna.

ALTER TABLE transacoes ADD COLUMN mes_ref DATE NULL AFTER data;

UPDATE transacoes SET mes_ref = DATE_FORMAT(data, '%Y-%m-01') WHERE mes_ref IS NULL;

ALTER TABLE transacoes MODIFY mes_ref DATE NOT NULL;

CREATE INDEX idx_transacoes_mes_ref ON transacoes (mes_ref);
