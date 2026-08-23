package web

import (
	"io"
	"net/http"
	"strconv"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/financas"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/importador"
	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/util"
)

// limite do upload: extrato de banco nao passa disso nem de longe,
// e sem teto qualquer upload grande derruba o servidor por memoria
const tamanhoMaximoUpload = 8 << 20 // 8 MB

func (s *Servidor) importar(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, tamanhoMaximoUpload)

	if err := r.ParseMultipartForm(tamanhoMaximoUpload); err != nil {
		erroJSON(w, http.StatusBadRequest, "upload invalido ou maior que 8 MB: "+err.Error())
		return
	}

	loArqu, loCabe, err := r.FormFile("arquivo")
	if err != nil {
		erroJSON(w, http.StatusBadRequest, "envie o arquivo no campo 'arquivo'")
		return
	}
	defer loArqu.Close()

	laDado, err := io.ReadAll(loArqu)
	if err != nil {
		erroJSON(w, http.StatusBadRequest, "erro ao ler o arquivo: "+err.Error())
		return
	}

	loFont := importador.NovoCSV(
		loCabe.Filename,
		laDado,
		r.FormValue("conta"),
		r.FormValue("pessoa"),
		importador.TipoArquivo(r.FormValue("tipo")),
	)

	laTran, err := loFont.Importar(r.Context())
	if err != nil {
		erroJSON(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	// categoriza pelas regras antes de gravar, para nao precisar de segundo passe
	lnCateg := 0
	if laRegr, err := s.repo.ListarRegras(r.Context()); err == nil {
		lnCateg = importador.NovoCategorizador(laRegr).AplicarEm(laTran)
	}

	// prever=1 devolve o que ACONTECERIA sem gravar nada — usado na tela de conferencia
	if r.FormValue("prever") == "1" {
		laPrev := make([]transacaoDTO, 0, len(laTran))
		for _, loTran := range laTran {
			laPrev = append(laPrev, paraDTO(loTran))
		}
		responderJSON(w, http.StatusOK, map[string]any{
			"previa":        true,
			"arquivo":       loCabe.Filename,
			"tipo":          string(loFont.Detectado),
			"total_linhas":  loFont.Linhas,
			"validas":       len(laTran),
			"com_erro":      len(loFont.Erros),
			"categorizadas": lnCateg,
			"erros":         loFont.Erros,
			"transacoes":    laPrev,
		})
		return
	}

	lnInse, lnDupl, err := s.repo.InserirLote(r.Context(), laTran)
	if err != nil {
		erroJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	loImpo := financas.Importacao{
		Arquivo:     loCabe.Filename,
		Fonte:       "csv",
		Conta:       r.FormValue("conta"),
		TotalLinhas: loFont.Linhas,
		Importadas:  lnInse,
		Duplicadas:  lnDupl,
		ComErro:     len(loFont.Erros),
	}
	if err := s.repo.RegistrarImportacao(r.Context(), &loImpo); err != nil {
		// a importacao ja gravou; falhar o registro de auditoria nao deve desfazer nada
		erroJSON(w, http.StatusOK, "importado, mas o registro de auditoria falhou: "+err.Error())
		return
	}

	responderJSON(w, http.StatusOK, map[string]any{
		"id":            loImpo.ID,
		"arquivo":       loImpo.Arquivo,
		"tipo":          string(loFont.Detectado),
		"total_linhas":  loFont.Linhas,
		"importadas":    lnInse,
		"duplicadas":    lnDupl,
		"com_erro":      len(loFont.Erros),
		"categorizadas": lnCateg,
		"erros":         loFont.Erros,
	})
}

func (s *Servidor) listarImportacoes(w http.ResponseWriter, r *http.Request) {
	lnLimi, _ := strconv.Atoi(r.URL.Query().Get("limite"))

	laImpo, err := s.repo.ListarImportacoes(r.Context(), lnLimi)
	if err != nil {
		erroJSON(w, http.StatusInternalServerError, err.Error())
		return
	}

	laDTO := make([]map[string]any, 0, len(laImpo))
	for _, loImpo := range laImpo {
		laDTO = append(laDTO, map[string]any{
			"id":           loImpo.ID,
			"arquivo":      loImpo.Arquivo,
			"fonte":        loImpo.Fonte,
			"conta":        loImpo.Conta,
			"total_linhas": loImpo.TotalLinhas,
			"importadas":   loImpo.Importadas,
			"duplicadas":   loImpo.Duplicadas,
			"com_erro":     loImpo.ComErro,
			"criado_em":    util.FormDtHr(loImpo.CriadoEm),
		})
	}

	responderJSON(w, http.StatusOK, map[string]any{"importacoes": laDTO})
}
