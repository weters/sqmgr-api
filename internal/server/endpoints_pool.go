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
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/weters/sqmgr-api/internal/model"
	"github.com/weters/sqmgr-api/internal/validator"
	"net/http"
	"strconv"
	"time"
)

const minJoinPasswordLength = 6
const validationErrorMessage = "There were one or more errors with your request"

func (s *Server) poolHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := mux.Vars(r)["token"]
		pool, err := s.model.PoolByToken(r.Context(), token)
		if err != nil {
			if err == sql.ErrNoRows {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		user := r.Context().Value(ctxUserKey).(*model.User)

		isMemberOf, err := user.IsMemberOf(r.Context(), pool)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if !isMemberOf {
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxPoolKey, pool)))
	})
}

func (s *Server) postPoolTokenEndpoint() http.HandlerFunc {
	type payload struct {
		Action string  `json:"action"`
		IDs    []int64 `json:"ids"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		user := r.Context().Value(ctxUserKey).(*model.User)

		if !user.IsAdminOf(r.Context(), pool) {
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		if r.Header.Get("Content-Type") != "application/json" {
			s.writeErrorResponse(w, http.StatusUnsupportedMediaType, nil)
			return
		}

		var resp payload
		if err := json.NewDecoder(r.Body).Decode(&resp); err != nil {
			s.writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		var err error
		switch resp.Action {
		case "lock":
			pool.SetLocks(time.Now())
			err = pool.Save(r.Context())
		case "unlock":
			pool.SetLocks(time.Time{})
			err = pool.Save(r.Context())
		case "reorderGrids":
			err = pool.SetGridsOrder(r.Context(), resp.IDs)
		default:
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("unsupported action %s", resp.Action))
			return
		}

		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, poolResponse{
			PoolJSON: pool.JSON(),
			IsAdmin:  true,
		})
	}
}

func (s *Server) getPoolTokenLogEndpoint() http.HandlerFunc {
	const defaultPerPage = 100
	const maxPerPage = 100

	type response struct {
		Logs []*model.PoolSquareLogJSON `json:"logs"`
		Total int64 `json:"total"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		user := r.Context().Value(ctxUserKey).(*model.User)

		if !user.IsAdminOf(r.Context(), pool) {
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		offset, _ := strconv.ParseInt(r.FormValue("offset"), 10, 64)
		if offset < 0 {
			offset = 0
		}

		limit, _ := strconv.Atoi(r.FormValue("limit"))
		if limit <= 0 {
			limit = defaultPerPage
		}

		if limit > maxPerPage {
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("limit cannot exceed %d", maxPerPage))
		}


		logs, err := pool.Logs(r.Context(), offset, limit)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		count, err := pool.LogsCount(r.Context())
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		logsJSON := make([]*model.PoolSquareLogJSON, len(logs))
		for i, log := range logs {
			logsJSON[i] = log.JSON()
		}

		s.writeJSONResponse(w, http.StatusOK, response{
			Logs:  logsJSON,
			Total: count,
		})
	}
}

func (s *Server) deletePoolTokenGridIDEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		id, _ := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

		grid, err := pool.GridByID(r.Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if err := grid.Delete(r.Context()); err != nil {
			if err == model.ErrLastGrid {
				s.writeErrorResponse(w, http.StatusBadRequest, errors.New("you cannot delete the last grid"))
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusNoContent, nil)
	}
}

func (s *Server) getPoolConfiguration() http.HandlerFunc {
	type keyDescription struct {
		Key model.GridType `json:"key"`
		Description string `json:"description"`
	}

	gridTypes := model.GridTypes()
	gridTypesSlice := make([]keyDescription, len(gridTypes))
	for i, gt := range gridTypes {
		gridTypesSlice[i] = keyDescription{
			Key:   gt,
			Description: gt.Description(),
		}
	}

	resp := struct {
		NameMaxLength int                        `json:"nameMaxLength"`
		NotesMaxLength int                       `json:"notesMaxLength"`
		TeamNameMaxLength int                    `json:"teamNameMaxLength"`
		PoolSquareStates []model.PoolSquareState `json:"poolSquareStates"`
		GridTypes []keyDescription               `json:"gridTypes"`
		MinJoinPasswordLength int `json:"minJoinPasswordLength"`
	}{
		NameMaxLength:     model.NameMaxLength,
		NotesMaxLength:    model.NotesMaxLength,
		TeamNameMaxLength: model.TeamNameMaxLength,
		PoolSquareStates: model.PoolSquareStates,
		GridTypes: gridTypesSlice,
		MinJoinPasswordLength: minJoinPasswordLength,
	}

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		logrus.WithError(err).Fatal("could not encode pool configuration")
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write(jsonResp); err != nil {
			logrus.WithError(err).Error("could not write response")
		}
	}
}

func (s *Server) postPoolEndpoint() http.HandlerFunc{
	type payload struct {
		Name string `json:"name"`
		GridType string `json:"gridType"`
		JoinPassword string `json:"joinPassword"`
		ConfirmJoinPassword string `json:"confirmJoinPassword"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(ctxUserKey).(*model.User)
		if !user.Can(model.UserActionCreatePool){
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		if r.Header.Get("Content-Type") != "application/json"	 {
			s.writeErrorResponse(w, http.StatusUnsupportedMediaType, nil)
			return
		}

		var data payload
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			s.writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		v := validator.New()
		name := v.Printable("Squares Pool Name", data.Name)
		gridType := v.GridType("Grid Configuration", data.GridType)
		password := v.Password("Join Password", data.JoinPassword, data.ConfirmJoinPassword, minJoinPasswordLength)

		if !v.OK() {
			s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
				Status:           statusError,
				Error:            validationErrorMessage,
				ValidationErrors: v.Errors,
			})
			return
		}

		pool, err := s.model.NewPool(r.Context(), user.ID, name, gridType, password)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusCreated, poolResponse{
			PoolJSON: pool.JSON(),
			IsAdmin:  true,
		})
	}
}

func (s *Server) getPoolTokenEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(ctxUserKey).(*model.User)
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		isAdminOf := user.IsAdminOf(r.Context(), pool)

		s.writeJSONResponse(w, http.StatusOK, poolResponse{
			PoolJSON: pool.JSON(),
			IsAdmin:  isAdminOf,
		})
	}
}

func (s *Server) getPoolTokenGridEndpoint() http.HandlerFunc {
	const defaultPerPage = 10
	const maxPerPage = 25

	type response struct {
		Grids []*model.GridJSON `json:"grids"`
		Total int64             `json:"total"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)

		offset, _ := strconv.ParseInt(r.FormValue("offset"), 10, 64)
		if offset < 0 {
			offset = 0
		}

		limit, _ := strconv.Atoi(r.FormValue("limit"))
		if limit < 1 {
			limit = defaultPerPage
		} else if limit > maxPerPage {
			s.writeErrorResponse(w, http.StatusBadGateway, fmt.Errorf("limit cannot exceed %d", maxPerPage))
			return
		}

		grids, err := pool.Grids(r.Context(), offset, limit)
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		count, err := pool.GridsCount(r.Context())
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		gridsJSON := make([]*model.GridJSON, len(grids))
		for i, grid := range grids {
			gridsJSON[i] = grid.JSON()
		}

		s.writeJSONResponse(w, http.StatusOK, response{
			Grids: gridsJSON,
			Total: count,
		})
	}
}

func (s *Server) getPoolTokenGridIDEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		id, _ := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)

		grid, err := pool.GridByID(r.Context(), id)
		if err != nil {
			if err == sql.ErrNoRows {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if err := grid.LoadSettings(r.Context()); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		s.writeJSONResponse(w, http.StatusOK, grid.JSON())
	}
}

func (s *Server) getPoolTokenSquareEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)

		squares, err := pool.Squares()
		if err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		squaresJSON := make(map[int]*model.PoolSquareJSON)
		for key, square := range squares {
			squaresJSON[key] = square.JSON()
		}

		s.writeJSONResponse(w, http.StatusOK, squaresJSON)
	}
}

func (s *Server) getPoolTokenSquareIDEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		user := r.Context().Value(ctxUserKey).(*model.User)

		squareID, _ := strconv.Atoi(mux.Vars(r)["id"])
		square, err := pool.SquareBySquareID(squareID)
		if err != nil {
			if err == sql.ErrNoRows {
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		if user.IsAdminOf(r.Context(), pool) {
			if err := square.LoadLogs(r.Context()); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		}

		s.writeJSONResponse(w, http.StatusOK, square.JSON())
	}
}

func (s *Server) postPoolTokenSquareIDEndpoint() http.HandlerFunc {
	type postPayload struct {
		Claimant string                `json:"claimant"`
		State    model.PoolSquareState `json:"state"`
		Note     string                `json:"note"`
		Unclaim  bool                  `json:"unclaim"`
		Rename   bool                  `json:"rename"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		user := r.Context().Value(ctxUserKey).(*model.User)
		squareID, _ := strconv.Atoi(mux.Vars(r)["id"])
		square, err := pool.SquareBySquareID(squareID)
		if err != nil {
			if err == sql.ErrNoRows {
				logrus.WithFields(logrus.Fields{
					"pool": pool.ID(),
					"square": squareID,
				}).Error("could not find square")
				s.writeErrorResponse(w, http.StatusNotFound, nil)
				return
			}

			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		lr := logrus.WithField("square-id", squareID)

		isAdmin := user.IsAdminOf(r.Context(), pool)

		// if the user isn't an admin and the grid is locked, do not let the user do anything
		if pool.IsLocked() && !isAdmin {
			s.writeErrorResponse(w, http.StatusForbidden, errors.New("The grid is locked"))
			return
		}

		dec := json.NewDecoder(r.Body)
		var payload postPayload
		if err := dec.Decode(&payload); err != nil {
			s.writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}



		if payload.Rename {
			if !isAdmin {
				s.writeErrorResponse(w, http.StatusForbidden, errors.New("only an admin can rename a square"))
				return
			}

			v := validator.New()
			claimant := v.Printable("name", payload.Claimant)
			claimant = v.ContainsWordChar("name", claimant)

			if claimant == square.Claimant {
				v.AddError("claimant", "must be a different name")
			}

			if !v.OK() {
				s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
					Status:           statusError,
					Error:            validationErrorMessage,
					ValidationErrors: v.Errors,
				})
				return
			}

			oldClaimant := square.Claimant
			square.Claimant = claimant
			lr.WithFields(logrus.Fields{
				"oldClaimant": oldClaimant,
				"claimant":    claimant,
			}).Info("renaming sqaure")

			if err := square.Save(r.Context(), true, model.PoolSquareLog{
				RemoteAddr: r.RemoteAddr,
				Note:       fmt.Sprintf("admin: changed claimant from %s", oldClaimant),
			}); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		} else if len(payload.Claimant) > 0 {
			// making a claim
			v := validator.New()
			claimant := v.Printable("name", payload.Claimant)
			claimant = v.ContainsWordChar("name", claimant)

			if !v.OK() {
				s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
					Status:           statusError,
					Error:            validationErrorMessage,
					ValidationErrors: v.Errors,
				})
				return
			}

			square.Claimant = claimant
			square.State = model.PoolSquareStateClaimed
			square.SetUserID(user.ID)

			lr.WithField("claimant", payload.Claimant).Info("claiming square")
			if err := square.Save(r.Context(), false, model.PoolSquareLog{
				RemoteAddr: r.RemoteAddr,
				Note:       "user: initial claim",
			}); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		} else if payload.Unclaim && square.UserID() == user.ID {
			// trying to unclaim as user
			square.State = model.PoolSquareStateUnclaimed
			square.SetUserID(user.ID)

			if err := square.Save(r.Context(), false, model.PoolSquareLog{
				RemoteAddr: r.RemoteAddr,
				Note:       fmt.Sprintf("user: `%s` unclaimed", square.Claimant),
			}); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		} else if isAdmin {
			// admin actions
			if payload.State.IsValid() {
				square.State = payload.State
			}

			if err := square.Save(r.Context(), true, model.PoolSquareLog{
				RemoteAddr: r.RemoteAddr,
				Note:       payload.Note,
			}); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		} else {
			lr.WithField("remoteAddr", r.RemoteAddr).Warn("non-admin tried to administer squares")
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		if isAdmin {
			if err := square.LoadLogs(r.Context()); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		}

		s.writeJSONResponse(w, http.StatusOK, square.JSON())
	}
}

func (s *Server) postPoolTokenGridIDEndpoint() http.HandlerFunc {
	type payload struct {
		Action string `json:"action"`
		Data   *struct {
			EventDate      string `json:"eventDate"`
			Notes          string `json:"notes"`
			HomeTeamName   string `json:"homeTeamName"`
			HomeTeamColor1 string `json:"homeTeamColor1"`
			HomeTeamColor2 string `json:"homeTeamColor2"`
			AwayTeamName   string `json:"awayTeamName"`
			AwayTeamColor1 string `json:"awayTeamColor1"`
			AwayTeamColor2 string `json:"awayTeamColor2"`
		} `json:"data,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		pool := r.Context().Value(ctxPoolKey).(*model.Pool)
		user := r.Context().Value(ctxUserKey).(*model.User)

		if !user.IsAdminOf(r.Context(), pool) {
			s.writeErrorResponse(w, http.StatusForbidden, nil)
			return
		}

		var data payload
		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&data); err != nil {
			s.writeErrorResponse(w, http.StatusBadRequest, err)
			return
		}

		gridID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
		if err != nil {
			panic(err)
		}

		var grid *model.Grid
		if gridID > 0 {
			var err error
			grid, err = pool.GridByID(r.Context(), gridID)
			if err != nil {
				if err == sql.ErrNoRows {
					s.writeErrorResponse(w, http.StatusNotFound, nil)
					return
				}

				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			if err := grid.LoadSettings(r.Context()); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}
		} else if data.Action != "save" {
			s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("cannot call action %s without an ID", data.Action))
			return
		}

		switch data.Action {
		case "drawNumbers":
			if err := grid.SelectRandomNumbers(); err != nil {
				if err == model.ErrNumbersAlreadyDrawn {
					s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("The numbers have already been drawn"))
					return
				}

				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			if err := grid.Save(r.Context()); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			s.writeJSONResponse(w, http.StatusOK, grid.JSON())
			return
		case "save":
			if data.Data == nil {
				s.writeErrorResponse(w, http.StatusBadRequest, errors.New("missing data in payload"))
				return
			}

			v := validator.New()
			eventDate := v.Datetime("Event Date", data.Data.EventDate, "00:00", "0", true)
			homeTeamName := v.Printable("Home Team Name", data.Data.HomeTeamName, true)
			homeTeamName = v.MaxLength("Home Team Name", homeTeamName, model.TeamNameMaxLength)
			homeTeamColor1 := v.Color("Home Team Colors", data.Data.HomeTeamColor1, true)
			homeTeamColor2 := v.Color("Home Team Colors", data.Data.HomeTeamColor2, true)
			awayTeamName := v.Printable("Away Team Name", data.Data.AwayTeamName, true)
			awayTeamName = v.MaxLength("Away Team Name", awayTeamName, model.TeamNameMaxLength)
			awayTeamColor1 := v.Color("Away Team Colors", data.Data.AwayTeamColor1, true)
			awayTeamColor2 := v.Color("Away Team Colors", data.Data.AwayTeamColor2, true)
			notes := v.PrintableWithNewline("Notes", data.Data.Notes, true)
			notes = v.MaxLength("Notes", notes, model.NotesMaxLength)

			if !v.OK() {
				s.writeJSONResponse(w, http.StatusBadRequest, ErrorResponse{
					Status:           statusError,
					Error:            "There were one or more validation errors",
					ValidationErrors: v.Errors,
				})
				return
			}

			if grid == nil {
				grid = pool.NewGrid()
			}

			grid.SetEventDate(eventDate)
			grid.SetHomeTeamName(homeTeamName)
			grid.SetAwayTeamName(awayTeamName)
			settings := grid.Settings()
			settings.SetNotes(notes)
			settings.SetHomeTeamColor1(homeTeamColor1)
			settings.SetHomeTeamColor2(homeTeamColor2)
			settings.SetAwayTeamColor1(awayTeamColor1)
			settings.SetAwayTeamColor2(awayTeamColor2)

			if err := grid.Save(r.Context()); err != nil {
				s.writeErrorResponse(w, http.StatusInternalServerError, err)
				return
			}

			s.writeJSONResponse(w, http.StatusAccepted, grid.JSON())
			return
		}

		s.writeErrorResponse(w, http.StatusBadRequest, fmt.Errorf("unsupported action %s", data.Action))
		return
	}
}

func (s *Server) postPoolTokenMemberEndpoint() http.HandlerFunc {
	type payload struct {
		Password string `json:"password"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(ctxUserKey).(*model.User)
		token := mux.Vars(r)["token"]
		pool, err := s.model.PoolByToken(r.Context(), token)
		if err != nil {
			if err == sql.ErrNoRows {
				s.writeErrorResponse(w,http.StatusNotFound, nil)
				return
			}

			s.writeErrorResponse(w,http.StatusInternalServerError, err)
			return
		}

		if r.Header.Get("Content-Type") != "application/json" {
			s.writeErrorResponse(w, http.StatusUnsupportedMediaType, nil)
			return
		}

		var data payload
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			s.writeErrorResponse(w,http.StatusBadRequest, err)
			return
		}

		if !pool.PasswordIsValid(data.Password) {
			s.writeErrorResponse(w, http.StatusBadRequest, errors.New("password is invalid"))
			return
		}

		if err := user.JoinPool(r.Context(), pool); err != nil {
			s.writeErrorResponse(w, http.StatusInternalServerError, err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

type poolResponse struct {
	*model.PoolJSON
	IsAdmin bool `json:"isAdmin"`
}
