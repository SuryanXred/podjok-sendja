package httpapi

import (
	"bytes"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"caffe-podjok-sendja/internal/staff"
	"caffe-podjok-sendja/internal/view"
)

const staffSessionCookieName = "pds_staff_session"

func getLoggedInStaff(c *gin.Context) *staff.StaffUser {
	token, err := c.Cookie(staffSessionCookieName)
	if err != nil || token == "" {
		return nil
	}
	return staff.DefaultStore.GetStaffBySession(token)
}

func setStaffSession(c *gin.Context, u *staff.StaffUser) {
	sess, err := staff.DefaultStore.CreateSession(u.ID)
	if err != nil {
		return
	}
	c.SetCookie(
		staffSessionCookieName,
		sess.Token,
		86400, // 24 jam shift pegawai
		"/",
		"",
		false,
		true,
	)
}

func clearStaffSession(c *gin.Context) {
	token, _ := c.Cookie(staffSessionCookieName)
	if token != "" {
		staff.DefaultStore.DeleteSession(token)
	}
	c.SetCookie(
		staffSessionCookieName,
		"",
		-1,
		"/",
		"",
		false,
		true,
	)
}

func staffLoginPageHandler(c *gin.Context) {
	if u := getLoggedInStaff(c); u != nil {
		if u.IsAdmin() {
			c.Redirect(http.StatusSeeOther, "/admin")
			return
		}
		c.Redirect(http.StatusSeeOther, "/kasir")
		return
	}

	redirectTo := strings.TrimSpace(c.Query("redirect"))
	msg := strings.TrimSpace(c.Query("msg"))

	var page bytes.Buffer
	if err := view.StaffLoginPage("", msg, redirectTo).Render(c.Request.Context(), &page); err != nil {
		c.String(http.StatusInternalServerError, "Gagal memuat halaman login pegawai.")
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", page.Bytes())
}

func staffLoginPostHandler(c *gin.Context) {
	identifier := strings.TrimSpace(c.PostForm("identifier"))
	password := c.PostForm("password")
	redirectTo := strings.TrimSpace(c.PostForm("redirect"))

	user, err := staff.DefaultStore.Authenticate(identifier, password)
	if err != nil {
		var page bytes.Buffer
		_ = view.StaffLoginPage(err.Error(), "", redirectTo).Render(c.Request.Context(), &page)
		c.Data(http.StatusUnprocessableEntity, "text/html; charset=utf-8", page.Bytes())
		return
	}

	setStaffSession(c, user)

	// Arahkan ke redirect target jika valid dan diizinkan perannya
	if redirectTo != "" && strings.HasPrefix(redirectTo, "/") {
		if strings.HasPrefix(redirectTo, "/admin") && !user.IsAdmin() {
			// Kasir tidak boleh diarahkan ke /admin
			c.Redirect(http.StatusSeeOther, "/kasir")
			return
		}
		c.Redirect(http.StatusSeeOther, redirectTo)
		return
	}

	if user.IsAdmin() {
		c.Redirect(http.StatusSeeOther, "/admin")
		return
	}
	c.Redirect(http.StatusSeeOther, "/kasir")
}

func staffLogoutHandler(c *gin.Context) {
	clearStaffSession(c)
	c.Redirect(http.StatusSeeOther, "/pegawai/masuk?msg=Anda+telah+berhasil+keluar+shift.")
}
