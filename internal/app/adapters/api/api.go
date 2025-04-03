package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/apache/synapse-go/internal/app/adapters/mediation"
	"github.com/apache/synapse-go/internal/app/core/domain"
	"github.com/apache/synapse-go/internal/pkg/core/synctx"
)

// type API struct{
// 	Config domain.APIConfig
// }

func InitializaRouter(ctx context.Context, config domain.APIConfig){
	router := http.NewServeMux()
	mediationEngine :=mediation.NewMediationEngine()
	for _, resource := range config.Resources {
		pattern := resource.Methods+ " " + config.Context + resource.URITemplate
		router.HandleFunc(pattern, func (w http.ResponseWriter, r *http.Request){
			msgContext := synctx.CreateMsgContext()
			//create msg context from r request
			mediationEngine.MediateAPIMessage(ctx, msgContext, resource.InSequence, resource.FaultSequence)
			fmt.Fprintf(w, "host, %s!", r.URL.Host)
		})
	}
	http.ListenAndServe(":8000", router)
}



