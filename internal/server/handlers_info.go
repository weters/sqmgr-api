/*
Copyright 2019 Tom Peters

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"fmt"
	"net/http"
)

func (s *Server) infoHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Session(r)
		user, err := session.LoggedInUser()
		session.Save()

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, "Session Values:")
		for key, value := range session.Values {
			fmt.Fprintf(w, "%s=%s\n", key, value)
		}

		fmt.Fprintln(w, "\nLogged in user:")

		if err != nil {
			fmt.Fprintf(w, "error getting logged in user: %v\n", err)
		} else {
			fmt.Fprintf(w, "email=%s\n", user.Email)
		}

		return
	}
}