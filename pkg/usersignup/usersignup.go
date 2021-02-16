package usersignup

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	toolchainv1alpha1 "github.com/codeready-toolchain/api/pkg/apis/toolchain/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
)

var (
	specialCharRegexp = regexp.MustCompile("[^A-Za-z0-9]")
	onlyNumbers       = regexp.MustCompile("^[0-9]*$")
)

func GenerateCompliantUsername(instance *toolchainv1alpha1.UserSignup, cl client.Client, forbiddenUsernamePrefixes []string ) (string, error) {
	replaced := TransformUsername(instance.Spec.Username)

	// Check for any forbidden prefixes
	for _, prefix := range forbiddenUsernamePrefixes {
		if strings.HasPrefix(replaced, prefix) {
			replaced = fmt.Sprintf("%s%s", "crt-", replaced)
			break
		}
	}

	validationErrors := validation.IsQualifiedName(replaced)
	if len(validationErrors) > 0 {
		return "", fmt.Errorf(fmt.Sprintf("transformed username [%s] is invalid", replaced))
	}

	transformed := replaced

	for i := 2; i < 101; i++ { // No more than 100 attempts to find a vacant name
		mur := &toolchainv1alpha1.MasterUserRecord{}
		// Check if a MasterUserRecord exists with the same transformed name
		namespacedMurName := types.NamespacedName{Namespace: instance.Namespace, Name: transformed}
		err := cl.Get(context.TODO(), namespacedMurName, mur)
		if err != nil {
			if !errors.IsNotFound(err) {
				return "", err
			}
			// If there was a NotFound error looking up the mur, it means we found an available name
			return transformed, nil
		} else if mur.Labels[toolchainv1alpha1.MasterUserRecordOwnerLabelKey] == instance.Name {
			// If the found MUR has the same UserID as the UserSignup, then *it* is the correct MUR -
			// Return an error here and allow the reconcile() function to pick it up on the next loop
			return "", fmt.Errorf(fmt.Sprintf("INFO: could not generate compliant username as MasterUserRecord with the same name [%s] and user id [%s] already exists. The next reconcile loop will pick it up.", mur.Name, instance.Name))
		}

		transformed = fmt.Sprintf("%s-%d", replaced, i)
	}

	return "", fmt.Errorf(fmt.Sprintf("unable to transform username [%s] even after 100 attempts", instance.Spec.Username))
}

func TransformUsername(username string) string {
	newUsername := specialCharRegexp.ReplaceAllString(strings.Split(username, "@")[0], "-")
	if len(newUsername) == 0 {
		newUsername = strings.ReplaceAll(username, "@", "at-")
	}
	newUsername = specialCharRegexp.ReplaceAllString(newUsername, "-")

	matched := onlyNumbers.MatchString(newUsername)
	if matched {
		newUsername = "crt-" + newUsername
	}
	for strings.Contains(newUsername, "--") {
		newUsername = strings.ReplaceAll(newUsername, "--", "-")
	}
	if strings.HasPrefix(newUsername, "-") {
		newUsername = "crt" + newUsername
	}
	if strings.HasSuffix(newUsername, "-") {
		newUsername = newUsername + "crt"
	}
	return newUsername
}