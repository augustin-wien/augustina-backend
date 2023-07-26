package handlers

import "net/http"

type ViewSet interface{
	List(w http.ResponseWriter, r *http.Request);
	Create(w http.ResponseWriter, r *http.Request);
	Retrieve(w http.ResponseWriter, r *http.Request);
	Update(w http.ResponseWriter, r *http.Request);
	Destroy(w http.ResponseWriter, r *http.Request)
}
