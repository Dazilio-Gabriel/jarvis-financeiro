package web

import (
	"net/http"
	"strings"

	"github.com/Dazilio-Gabriel/jarvis-financeiro/internal/armazenamento"
)

// ehAdmin: o painel do sistema e so do dono (DONO_EMAIL). Sem DONO_EMAIL, ninguem e admin.
func (s *Servidor) ehAdmin(poUsua armazenamento.Usuario) bool {
	return s.donoMail != "" && strings.EqualFold(poUsua.Email, s.donoMail)
}

// exigirAdmin vai depois do exigirLogin: logado mas nao dono recebe 404, e nao 403, para nem
// confirmar que a rota existe
func (s *Servidor) exigirAdmin(poProx http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.ehAdmin(usuarioDe(r)) {
			erroJSON(w, http.StatusNotFound, "rota inexistente")
			return
		}
		poProx(w, r)
	}
}

type pontoDTO struct {
	Dia   string `json:"dia"`
	Valor int    `json:"valor"`
}

func pontos(paPont []armazenamento.PontoDia) []pontoDTO {
	laDTO := make([]pontoDTO, 0, len(paPont))
	for _, loPont := range paPont {
		laDTO = append(laDTO, pontoDTO{Dia: loPont.Dia, Valor: loPont.Valor})
	}
	return laDTO
}

// painelAdmin: numeros da operacao. Nenhum valor financeiro de familia sai daqui (LGPD).
func (s *Servidor) painelAdmin(w http.ResponseWriter, r *http.Request) {
	loPain, err := s.auth.PainelAdmin(r.Context())
	if err != nil {
		erroInterno(w, err)
		return
	}

	laUsua := make([]map[string]any, 0, len(loPain.Lista))
	for _, loLinh := range loPain.Lista {
		var lcUlti string
		if loLinh.UltimoAcesso != nil {
			lcUlti = loLinh.UltimoAcesso.Format("2006-01-02T15:04")
		}
		laUsua = append(laUsua, map[string]any{
			"nome":          loLinh.Nome,
			"email":         loLinh.Email,
			"familia":       loLinh.Familia,
			"criado_em":     loLinh.CriadoEm.Format("2006-01-02"),
			"ultimo_acesso": lcUlti,
			"google":        loLinh.Google,
			"perguntas_30":  loLinh.Perguntas30,
		})
	}

	responderJSON(w, http.StatusOK, map[string]any{
		"usuarios":          loPain.Usuarios,
		"familias":          loPain.Familias,
		"com_google":        loPain.ComGoogle,
		"novos_7":           loPain.Novos7,
		"novos_30":          loPain.Novos30,
		"ativos_7":          loPain.Ativos7,
		"transacoes":        loPain.Transacoes,
		"importacoes_30":    loPain.Importacoes30,
		"perguntas_hoje":    loPain.PerguntasHoje,
		"tokens_hoje":       loPain.TokensHoje,
		"perguntas_30":      loPain.Perguntas30,
		"tokens_30":         loPain.Tokens30,
		"cadastros_por_dia": pontos(loPain.CadastrosPorDia),
		"perguntas_por_dia": pontos(loPain.PerguntasPorDia),
		"lista":             laUsua,
		"chat_ligado":       s.assis != nil,
		"google_ligado":     s.google != nil,
	})
}
