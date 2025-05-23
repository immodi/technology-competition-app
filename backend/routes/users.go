package routes

import (
	"database/sql"
	"encoding/json"
	"immodi/submission-backend/helpers"
	"immodi/submission-backend/repos"
	"immodi/submission-backend/routes/requests"
	"immodi/submission-backend/routes/responses"
	helper_structs "immodi/submission-backend/structs"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func UsersRouter(r chi.Router, db *sql.DB, api *helper_structs.API) {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		helpers.ProtectedHandler(w, r, func(username string) bool {
			return api.UserRepo.IsAdmin(username)
		}, GetAllUsers(api.UserRepo))
	})
	r.Put("/", func(w http.ResponseWriter, r *http.Request) {
		helpers.ProtectedHandler(w, r, func(username string) bool {
			return api.UserRepo.IsAdmin(username)
		}, UpdateUserRole(api.UserRepo))
	})
	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		helpers.ProtectedHandler(w, r, func(username string) bool {
			userId, err := helpers.ParseUserIdFromRoute(r)
			if err != nil {
				return false
			}
			return api.UserRepo.IsSameUser(username, userId) || api.UserRepo.IsAdmin(username)
		}, GetUser(api.UserRepo))
	})

	r.Get("/data", func(w http.ResponseWriter, r *http.Request) {
		helpers.ProtectedHandler(w, r, nil, GetUserDataFromToken(api.UserRepo))
	})

	r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
		helpers.ProtectedHandler(w, r, func(username string) bool {
			userId, err := helpers.ParseUserIdFromRoute(r)
			if err != nil {
				return false
			}
			return api.UserRepo.IsSameUser(username, userId) || api.UserRepo.IsAdmin(username)
		}, DeleteUser(api.UserRepo))
	})

	r.Get("/events/{id}", func(w http.ResponseWriter, r *http.Request) {
		helpers.ProtectedHandler(w, r, func(username string) bool {
			userId, err := helpers.ParseUserIdFromRoute(r)
			if err != nil {
				return false
			}
			return api.UserRepo.IsSameUser(username, userId) || api.UserRepo.IsAdmin(username)
		}, GetUserEvents(api.EventRepo))
	})
}

func GetAllUsers(userRepo repos.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := userRepo.GetAllUsers()
		if err != nil {
			helpers.HttpError(w, http.StatusInternalServerError, "Failed to get all users")
			return
		}

		helpers.HttpJson(w, http.StatusOK, users)
	}
}

func GetUser(userRepo repos.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			helpers.HttpError(w, http.StatusBadRequest, "Invalid user ID, pass a valid one")
			return
		}

		user, err := userRepo.GetUserById(id)
		if err != nil {
			helpers.HttpError(w, http.StatusInternalServerError, "Could not retrieve user")
			return
		}
		if user == nil {
			helpers.HttpError(w, http.StatusNotFound, "User not found")
			return
		}

		response := &responses.UserResponse{
			UserId:    user.ID,
			Role:      user.Role,
			Username:  user.Username,
			CreatedAt: user.CreatedAt,
			Tickets:   user.Tickets,
		}
		helpers.HttpJson(w, http.StatusOK, response)
	}
}

func GetUserDataFromToken(userRepo repos.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, err := helpers.GetUserNameFromToken(r)
		if err != nil {
			helpers.HttpError(w, http.StatusInternalServerError, "couldn't retrieve user")
		}

		user, err := userRepo.GetUserByUsername(username)
		if err != nil {
			helpers.HttpError(w, http.StatusInternalServerError, "could not retrieve user")
			return
		}
		if user == nil {
			helpers.HttpError(w, http.StatusNotFound, "user not found, try again later")
			return
		}

		response := &responses.UserResponse{
			UserId:    user.ID,
			Role:      user.Role,
			Username:  user.Username,
			CreatedAt: user.CreatedAt,
			Tickets:   user.Tickets,
		}
		helpers.HttpJson(w, http.StatusOK, response)
	}
}

func DeleteUser(userRepo repos.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			helpers.HttpError(w, http.StatusBadRequest, "Invalid user ID, pass a valid one")
			return
		}

		err = userRepo.DeleteUser(id)
		if err != nil {
			helpers.HttpError(w, http.StatusInternalServerError, "Could not delete the user")
			return
		}

		res := &responses.UserDeletionResponse{
			Message: "User deleted successfully",
		}

		helpers.HttpJson(w, http.StatusOK, res)
	}
}

func UpdateUserRole(userRepo repos.UserInterface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req requests.UserRoleUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			helpers.HttpError(w, http.StatusBadRequest, "Invalid request")
			return
		}
		if req.UserId == 0 || req.Role == "" {
			helpers.HttpError(w, http.StatusBadRequest, "Missing userId and role")
			return
		}

		if req.Role != "admin" && req.Role != "user" {
			helpers.HttpError(w, http.StatusBadRequest, "invalid user role, role must be 'admin' or 'user'")
			return
		}

		err := userRepo.UpdateUserRole(req.UserId, req.Role)
		if err != nil {
			helpers.HttpError(w, http.StatusInternalServerError, "failed to update user role")
			return
		}

		user, err := userRepo.GetUserById(req.UserId)
		if err != nil {
			helpers.HttpError(w, http.StatusInternalServerError, "User update succeeded but fetch failed")
			return
		}

		res := &responses.UserResponse{
			UserId:    user.ID,
			Role:      user.Role,
			Username:  user.Username,
			CreatedAt: user.CreatedAt,
		}

		helpers.HttpJson(w, http.StatusOK, res)
	}
}

func GetUserEvents(eventRepository *repos.EventRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			helpers.HttpError(w, http.StatusBadRequest, "invalid user ID, pass a valid one")
			return
		}

		events, err := eventRepository.GetEventsForUser(id)
		if err != nil {
			helpers.HttpError(w, http.StatusInternalServerError, "failed to get user events")
			return
		}

		helpers.HttpJson(w, http.StatusOK, events)
	}
}
