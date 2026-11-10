-- Cores de categoria no visual novo: a familia de tons do painel JuditeVendas, com o laranja exato
-- da marca, validada para daltonismo nos dois temas. Gravado fica o tom do tema claro e a tela troca
-- pelo do tema escuro. O vermelho saiu das padrao (ele quer dizer "estourou o orcamento"), entao
-- Saude vira azul-petroleo. Outros, Entrada e "sem categoria" viram cinzas quentes do tema novo.
-- Um UPDATE so, com CASE sobre o valor original: uma troca nunca pega o resultado de outra.
-- Se Saude cair na mesma cor de uma categoria criada pelo usuario, o NormalizarCores da subida
-- da outra cor para a mais nova.
UPDATE categorias SET cor = CASE LOWER(cor)
    WHEN '#3987e5' THEN '#2a78d6'
    WHEN '#d95926' THEN '#e2552d'
    WHEN '#199e70' THEN '#1baf7a'
    WHEN '#c98500' THEN '#eda100'
    WHEN '#d55181' THEN '#e87ba4'
    WHEN '#9085e9' THEN '#6250d6'
    WHEN '#e66767' THEN '#0f9bb0'
    WHEN '#8b9490' THEN '#a1978f'
    WHEN '#7fa59a' THEN '#7d8b98'
    WHEN '#5b6660' THEN '#c9c1b9'
END
WHERE LOWER(cor) IN ('#3987e5', '#d95926', '#199e70', '#c98500', '#d55181', '#9085e9', '#e66767',
                     '#8b9490', '#7fa59a', '#5b6660');
