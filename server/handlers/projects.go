package handlers

import (
	"encoding/json"
	"net/http"
)

func (handler *ServerHandler) ListProject(w http.ResponseWriter, r *http.Request) {

	if validAuth, err := VerifyToken(r); !validAuth {
		http.Error(w, err, 400)
		return
	}

	applications, err := handler.Repo.GetApplicationsFromDB()
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if len(*applications) == 0 {
		http.Error(w, "There are no apps currently!", 400)
		return
	}

	json.NewEncoder(w).Encode(applications)

}
