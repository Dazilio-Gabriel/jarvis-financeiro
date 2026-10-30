-- Cores das categorias numa paleta validada (scripts do dataviz: separacao para daltonismo,
-- distancia minima entre vizinhas e contraste >= 3:1 sobre o fundo escuro #0e1411).
-- Antes, Mercado e Entrada eram o mesmo verde e varias categorias ficavam iguais nos graficos.
-- Categorias de gasto: um tom cada. Outros / Entrada: neutros (nao competem com as de gasto).

UPDATE categorias SET cor = '#3987e5' WHERE nome = 'Mercado';
UPDATE categorias SET cor = '#d95926' WHERE nome = 'Alimentação';
UPDATE categorias SET cor = '#199e70' WHERE nome = 'Transporte';
UPDATE categorias SET cor = '#c98500' WHERE nome = 'Moradia';
UPDATE categorias SET cor = '#d55181' WHERE nome = 'Lazer';
UPDATE categorias SET cor = '#008300' WHERE nome = 'Educação';
UPDATE categorias SET cor = '#9085e9' WHERE nome = 'Assinaturas';
UPDATE categorias SET cor = '#e66767' WHERE nome = 'Saúde';
UPDATE categorias SET cor = '#8b9490' WHERE nome = 'Outros';
UPDATE categorias SET cor = '#7fa59a' WHERE nome = 'Entrada';

-- categorias criadas pelo usuario (ex.: Jogos) recebem o primeiro tom extra; as proximas o
-- servidor escolhe entre os que a familia ainda nao usa
UPDATE categorias SET cor = '#0f9bb0'
WHERE nome NOT IN ('Mercado', 'Alimentação', 'Transporte', 'Moradia', 'Lazer', 'Educação',
                   'Assinaturas', 'Saúde', 'Outros', 'Entrada');
