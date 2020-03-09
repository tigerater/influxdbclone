package main

import (
	"context"
	"errors"
	"os"

	platform "github.com/influxdata/influxdb"
	"github.com/influxdata/influxdb/cmd/influx/internal"
	"github.com/influxdata/influxdb/http"
	"github.com/spf13/cobra"
)

func userCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "User management commands",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Usage()
		},
	}
	cmd.AddCommand(
		userCreateCmd(),
		userDeleteCmd(),
		userFindCmd(),
		userUpdateCmd(),
	)

	return cmd
}

var userUpdateFlags struct {
	id   string
	name string
}

func userUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update user",
		RunE:  wrapCheckSetup(userUpdateF),
	}

	cmd.Flags().StringVarP(&userUpdateFlags.id, "id", "i", "", "The user ID (required)")
	cmd.Flags().StringVarP(&userUpdateFlags.name, "name", "n", "", "The user name")
	cmd.MarkFlagRequired("id")

	return cmd
}

func newUserService() (platform.UserService, error) {
	if flags.local {
		return newLocalKVService()
	}
	return &http.UserService{
		Addr:               flags.host,
		Token:              flags.token,
		InsecureSkipVerify: flags.skipVerify,
	}, nil
}

func newUserResourceMappingService() (platform.UserResourceMappingService, error) {
	if flags.local {
		return newLocalKVService()
	}
	return &http.UserResourceMappingService{
		Addr:               flags.host,
		Token:              flags.token,
		InsecureSkipVerify: flags.skipVerify,
	}, nil
}

func userUpdateF(cmd *cobra.Command, args []string) error {
	s, err := newUserService()
	if err != nil {
		return err
	}

	var id platform.ID
	if err := id.DecodeFromString(userUpdateFlags.id); err != nil {
		return err
	}

	update := platform.UserUpdate{}
	if userUpdateFlags.name != "" {
		update.Name = &userUpdateFlags.name
	}

	user, err := s.UpdateUser(context.Background(), id, update)
	if err != nil {
		return err
	}

	w := internal.NewTabWriter(os.Stdout)
	w.WriteHeaders(
		"ID",
		"Name",
	)
	w.Write(map[string]interface{}{
		"ID":   user.ID.String(),
		"Name": user.Name,
	})
	w.Flush()

	return nil
}

var userCreateFlags struct {
	name     string
	password string
	orgID    string
}

func userCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create user",
		RunE:  wrapCheckSetup(userCreateF),
	}

	cmd.Flags().StringVarP(&userCreateFlags.name, "name", "n", "", "The user name (required)")
	cmd.MarkFlagRequired("name")
	cmd.Flags().StringVarP(&userCreateFlags.password, "password", "p", "", "The user password")
	cmd.Flags().StringVarP(&userCreateFlags.orgID, "org-id", "o", "", "The organization id the user belongs too. Is required if password provided.")

	return cmd
}

func userCreateF(cmd *cobra.Command, args []string) error {
	s, err := newUserService()
	if err != nil {
		return err
	}

	user := &platform.User{
		Name: userCreateFlags.name,
	}

	if err := s.CreateUser(context.Background(), user); err != nil {
		return err
	}

	writeOutput := func(headers []string, vals ...string) error {
		if len(headers) != len(vals) {
			return errors.New("invalid headers and val setup for writer")
		}

		m := make(map[string]interface{})
		for i, h := range headers {
			m[h] = vals[i]
		}
		w := internal.NewTabWriter(os.Stdout)
		w.WriteHeaders(headers...)
		w.Write(m)
		w.Flush()

		return nil
	}

	orgIDStr := userCreateFlags.orgID
	pass := userCreateFlags.password
	if orgIDStr == "" && pass == "" {
		return writeOutput([]string{"ID", "Name"}, user.ID.String(), user.Name)
	}

	if pass != "" && orgIDStr == "" {
		return errors.New("an org id is required when providing a user password")
	}

	orgID, err := platform.IDFromString(orgIDStr)
	if err != nil {
		return errors.New("an invalid org ID provided: " + orgIDStr)
	}

	userResMapSVC := &http.UserResourceMappingService{
		Addr:               flags.host,
		Token:              flags.token,
		InsecureSkipVerify: flags.skipVerify,
	}

	err = userResMapSVC.CreateUserResourceMapping(context.Background(), &platform.UserResourceMapping{
		UserID:       user.ID,
		UserType:     platform.Member,
		ResourceType: platform.OrgsResourceType,
		ResourceID:   *orgID,
	})
	if err != nil {
		return err
	}

	passSVC := &http.PasswordService{
		Addr:               flags.host,
		Token:              flags.token,
		InsecureSkipVerify: flags.skipVerify,
	}

	ctx := context.Background()
	if err := passSVC.SetPassword(ctx, user.ID, pass); err != nil {
		return err
	}

	return writeOutput([]string{"ID", "Name", "Organization ID"}, user.ID.String(), user.Name, orgID.String())
}

var userFindFlags struct {
	id   string
	name string
}

func userFindCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "find",
		Short: "Find user",
		RunE:  wrapCheckSetup(userFindF),
	}

	cmd.Flags().StringVarP(&userFindFlags.id, "id", "i", "", "The user ID")
	cmd.Flags().StringVarP(&userFindFlags.name, "name", "n", "", "The user name")

	return cmd
}

func userFindF(cmd *cobra.Command, args []string) error {
	s, err := newUserService()
	if err != nil {
		return err
	}

	filter := platform.UserFilter{}
	if userFindFlags.name != "" {
		filter.Name = &userFindFlags.name
	}
	if userFindFlags.id != "" {
		id, err := platform.IDFromString(userFindFlags.id)
		if err != nil {
			return err
		}
		filter.ID = id
	}

	users, _, err := s.FindUsers(context.Background(), filter)
	if err != nil {
		return err
	}

	w := internal.NewTabWriter(os.Stdout)
	w.WriteHeaders(
		"ID",
		"Name",
	)
	for _, u := range users {
		w.Write(map[string]interface{}{
			"ID":   u.ID.String(),
			"Name": u.Name,
		})
	}
	w.Flush()

	return nil
}

var userDeleteFlags struct {
	id string
}

func userDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete user",
		RunE:  wrapCheckSetup(userDeleteF),
	}

	cmd.Flags().StringVarP(&userDeleteFlags.id, "id", "i", "", "The user ID (required)")
	cmd.MarkFlagRequired("id")

	return cmd
}

func userDeleteF(cmd *cobra.Command, args []string) error {
	s, err := newUserService()
	if err != nil {
		return err
	}

	var id platform.ID
	if err := id.DecodeFromString(userDeleteFlags.id); err != nil {
		return err
	}

	ctx := context.Background()
	u, err := s.FindUserByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.DeleteUser(ctx, id); err != nil {
		return err
	}

	w := internal.NewTabWriter(os.Stdout)
	w.WriteHeaders(
		"ID",
		"Name",
		"Deleted",
	)
	w.Write(map[string]interface{}{
		"ID":      u.ID.String(),
		"Name":    u.Name,
		"Deleted": true,
	})
	w.Flush()

	return nil
}
