package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
	"url-shortener/internal/config"
	"url-shortener/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

// --------------------------------------------------------------------------
// Config builders — built lazily so env is loaded first
// --------------------------------------------------------------------------

func googleOAuthConfig() *oauth2.Config {
	base := strings.TrimRight(os.Getenv("SERVER_URL"), "/")
	return &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  base + "/auth/google/callback",
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint:     google.Endpoint,
	}
}

func githubOAuthConfig() *oauth2.Config {
	base := strings.TrimRight(os.Getenv("SERVER_URL"), "/")
	return &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		RedirectURL:  base + "/auth/github/callback",
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}
}

// --------------------------------------------------------------------------
// Helpers
// --------------------------------------------------------------------------

const oauthStateValue = "jugaadlink-oauth-state"

func mintJWT(userID uint) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}

func frontendCallbackURL(token string) string {
	base := strings.TrimRight(os.Getenv("FRONTEND_URL"), "/")
	return base + "/oauth/callback?token=" + token
}

// upsertOAuthUser finds an existing OAuth user or creates a new one.
func upsertOAuthUser(email, name, avatarURL, provider, providerID string) (*models.User, error) {
	var user models.User

	// Try to find by oauth provider + id first
	err := config.DB.Where("oauth_provider = ? AND oauth_id = ?", provider, providerID).First(&user).Error
	if err == nil {
		return &user, nil
	}

	// Try to find by email (existing email-signup user)
	err = config.DB.Where("email = ?", email).First(&user).Error
	if err == nil {
		// Link the OAuth provider to the existing account
		config.DB.Model(&user).Updates(models.User{
			OAuthProvider: provider,
			OAuthID:       providerID,
			AvatarURL:     avatarURL,
		})
		return &user, nil
	}

	// Create a brand-new user
	user = models.User{
		Email:         email,
		Name:          name,
		AvatarURL:     avatarURL,
		OAuthProvider: provider,
		OAuthID:       providerID,
	}
	if result := config.DB.Create(&user); result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// --------------------------------------------------------------------------
// Google
// --------------------------------------------------------------------------

func GoogleOAuthLogin(c *gin.Context) {
	cfg := googleOAuthConfig()
	url := cfg.AuthCodeURL(oauthStateValue, oauth2.AccessTypeOffline)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func GoogleOAuthCallback(c *gin.Context) {
	if c.Query("state") != oauthStateValue {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state"})
		return
	}

	cfg := googleOAuthConfig()
	oauthToken, err := cfg.Exchange(context.Background(), c.Query("code"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to exchange token: " + err.Error()})
		return
	}

	resp, err := cfg.Client(context.Background(), oauthToken).Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user info"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var info struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		Picture   string `json:"picture"`
		Verified  bool   `json:"verified_email"`
	}
	if err := json.Unmarshal(body, &info); err != nil || info.Email == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse user info"})
		return
	}

	user, err := upsertOAuthUser(info.Email, info.Name, info.Picture, "google", info.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upsert user"})
		return
	}

	jwtToken, err := mintJWT(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mint token"})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, frontendCallbackURL(jwtToken))
}

// --------------------------------------------------------------------------
// GitHub
// --------------------------------------------------------------------------

func GithubOAuthLogin(c *gin.Context) {
	cfg := githubOAuthConfig()
	url := cfg.AuthCodeURL(oauthStateValue)
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func GithubOAuthCallback(c *gin.Context) {
	if c.Query("state") != oauthStateValue {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid oauth state"})
		return
	}

	cfg := githubOAuthConfig()
	oauthToken, err := cfg.Exchange(context.Background(), c.Query("code"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to exchange token: " + err.Error()})
		return
	}

	client := cfg.Client(context.Background(), oauthToken)

	// Fetch public profile
	profileResp, err := client.Get("https://api.github.com/user")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch github profile"})
		return
	}
	defer profileResp.Body.Close()

	profileBody, _ := io.ReadAll(profileResp.Body)
	var profile struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	json.Unmarshal(profileBody, &profile)

	// GitHub may return an empty public email — fall back to the emails API
	email := profile.Email
	if email == "" {
		emailResp, err := client.Get("https://api.github.com/user/emails")
		if err == nil {
			defer emailResp.Body.Close()
			emailBody, _ := io.ReadAll(emailResp.Body)
			var emails []struct {
				Email    string `json:"email"`
				Primary  bool   `json:"primary"`
				Verified bool   `json:"verified"`
			}
			json.Unmarshal(emailBody, &emails)
			for _, e := range emails {
				if e.Primary && e.Verified {
					email = e.Email
					break
				}
			}
		}
	}

	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no verified email found on GitHub account"})
		return
	}

	name := profile.Name
	if name == "" {
		name = profile.Login
	}

	user, err := upsertOAuthUser(email, name, profile.AvatarURL, "github", fmt.Sprintf("%d", profile.ID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upsert user"})
		return
	}

	jwtToken, err := mintJWT(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mint token"})
		return
	}

	c.Redirect(http.StatusTemporaryRedirect, frontendCallbackURL(jwtToken))
}
