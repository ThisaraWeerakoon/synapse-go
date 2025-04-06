package api

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/apache/synapse-go/internal/app/adapters/mediation"
	"github.com/apache/synapse-go/internal/app/core/domain"
	"github.com/apache/synapse-go/internal/pkg/core/synctx"
)

// type API struct{
// 	Config domain.APIConfig
// }

func InitializeRouter(ctx context.Context, config domain.APIConfig){
	router := http.NewServeMux()
	mediationEngine :=mediation.NewMediationEngine()
	for _, resource := range config.Resources {
		pattern := resource.Methods+ " " + config.Context + resource.URITemplate
		router.HandleFunc(pattern, func (w http.ResponseWriter, r *http.Request){
			msgContext := synctx.CreateMsgContext()
			//create msg context from r request
			msgContext.Properties["request"] = r.URL.Path
			msgContext.Properties["method"] = r.Method
			msgContext.Headers["Content-Type"] = r.Header.Get("Content-Type")
			// Access request body
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusInternalServerError)
				return
			}
			msgContext.Message.RawPayload = body
			msgContext.Message.ContentType = r.Header.Get("Content-Type")



			mediationEngine.MediateAPIMessage(ctx, msgContext, resource.InSequence, resource.FaultSequence)
			fmt.Fprintf(w, "host, %s!", r.URL.Host)
		})
	}
	http.ListenAndServe(":8000", router)
}



