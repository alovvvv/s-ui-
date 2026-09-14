package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/alireza0/s-ui/config"
	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/service"
)

// confirm asks before an irreversible change. It returns false when there is no
// terminal to ask on, so a script that reaches this by accident stops rather
// than proceeding unattended.
func confirm(prompt string) bool {
	fmt.Print(prompt)
	answer, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		fmt.Println()
		return false
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}

// resetAdmin puts the credentials back to admin/admin.
//
// assumeYes comes from -yes. Without a prompt, a mistyped command turned a
// live panel into one with the best-known default credentials in the world,
// reachable from the internet, with nothing printed that looked like a
// warning.
func resetAdmin(assumeYes bool) {
	if !assumeYes {
		fmt.Println("This resets the first admin account to the username \"admin\" and the password \"admin\".")
		fmt.Println("Anyone who can reach the panel will be able to log in until you change it.")
		if !confirm("Type y to continue: ") {
			fmt.Println("cancelled")
			return
		}
	}

	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}

	userService := service.UserService{}
	err = userService.UpdateFirstUser("admin", "admin")
	if err != nil {
		fmt.Println("reset admin credentials failed:", err)
		return
	}
	fmt.Println("reset admin credentials success")
	fmt.Println("Change them now: s-ui admin -username <user> -password <pass>")
}

func updateAdmin(username string, password string) {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}

	if username != "" || password != "" {
		userService := service.UserService{}
		err := userService.UpdateFirstUser(username, password)
		if err != nil {
			fmt.Println("reset admin credentials failed:", err)
		} else {
			fmt.Println("reset admin credentials success")
		}
	}
}

func showAdmin() {
	err := database.InitDB(config.GetDBPath())
	if err != nil {
		fmt.Println(err)
		return
	}
	userService := service.UserService{}
	userModel, err := userService.GetFirstUser()
	if err != nil {
		fmt.Println("get current user info failed,error info:", err)
	}
	username := userModel.Username
	userpasswd := userModel.Password
	if (username == "") || (userpasswd == "") {
		fmt.Println("current username or password is empty")
	}
	fmt.Println("First admin credentials:")
	fmt.Println("\tUsername:\t", username)
	fmt.Println("\tPassword:\t <hashed, not recoverable>")
	fmt.Println("To set a new password, run: s-ui admin -username <user> -password <pass>")
}
