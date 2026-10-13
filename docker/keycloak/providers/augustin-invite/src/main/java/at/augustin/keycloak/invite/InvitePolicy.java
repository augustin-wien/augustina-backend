package at.augustin.keycloak.invite;

import org.keycloak.models.ClientModel;
import org.keycloak.models.Constants;
import org.keycloak.models.GroupModel;
import org.keycloak.models.RealmModel;
import org.keycloak.models.RoleModel;
import org.keycloak.models.UserModel;
import org.keycloak.models.credential.PasswordCredentialModel;

import java.util.List;

/**
 * Who may be signed in by an invite link, and who may set a password through
 * it. Checked when the link is issued and again when it is opened.
 */
final class InvitePolicy {

    /** Top-level group the backend puts every buyer of a digital item into. */
    private static final String CUSTOMER_GROUP = "customer";

    /** Realm roles of accounts that must always sign in with their own credentials. */
    private static final List<String> PRIVILEGED_REALM_ROLES = List.of("admin", "backoffice");

    /** A vendor who is also a customer is signed in, but sets no password here. */
    private static final String VENDOR_REALM_ROLE = "vendor";

    private InvitePolicy() {
    }

    /** A customer without any backoffice or Keycloak admin rights. */
    static boolean isInvitable(RealmModel realm, UserModel user) {
        return user.isEnabled() && isCustomer(user) && !isPrivileged(realm, user);
    }

    /** Invite links never replace a password, they only let a customer set the first one. */
    static boolean maySetPassword(RealmModel realm, UserModel user) {
        return isInvitable(realm, user)
            && !hasRealmRole(realm, user, VENDOR_REALM_ROLE)
            && !user.credentialManager().isConfiguredFor(PasswordCredentialModel.TYPE);
    }

    private static boolean isCustomer(UserModel user) {
        return user.getGroupsStream().anyMatch(InvitePolicy::isInCustomerGroup);
    }

    private static boolean isInCustomerGroup(GroupModel group) {
        GroupModel top = group;
        while (top.getParent() != null) {
            top = top.getParent();
        }
        return CUSTOMER_GROUP.equals(top.getName());
    }

    private static boolean hasRealmRole(RealmModel realm, UserModel user, String name) {
        RoleModel role = realm.getRole(name);
        return role != null && user.hasRole(role);
    }

    private static boolean isPrivileged(RealmModel realm, UserModel user) {
        for (String name : PRIVILEGED_REALM_ROLES) {
            if (hasRealmRole(realm, user, name)) {
                return true;
            }
        }
        ClientModel realmManagement = realm.getClientByClientId(Constants.REALM_MANAGEMENT_CLIENT_ID);
        return realmManagement != null && realmManagement.getRolesStream().anyMatch(user::hasRole);
    }
}
