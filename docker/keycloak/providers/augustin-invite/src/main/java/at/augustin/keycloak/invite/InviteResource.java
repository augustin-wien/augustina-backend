package at.augustin.keycloak.invite;

import com.fasterxml.jackson.databind.JsonNode;
import jakarta.ws.rs.Consumes;
import jakarta.ws.rs.ForbiddenException;
import jakarta.ws.rs.NotAuthorizedException;
import jakarta.ws.rs.POST;
import jakarta.ws.rs.Path;
import jakarta.ws.rs.Produces;
import jakarta.ws.rs.core.MediaType;
import jakarta.ws.rs.core.Response;
import org.keycloak.common.util.Time;
import org.keycloak.models.ClientModel;
import org.keycloak.models.KeycloakSession;
import org.keycloak.models.KeycloakUriInfo;
import org.keycloak.models.RealmModel;
import org.keycloak.models.UserModel;
import org.keycloak.protocol.oidc.utils.RedirectUtils;
import org.keycloak.services.managers.AppAuthManager;
import org.keycloak.services.managers.AuthenticationManager.AuthResult;
import org.keycloak.services.resources.LoginActionsService;
import org.keycloak.services.resources.admin.AdminAuth;
import org.keycloak.services.resources.admin.fgap.AdminPermissions;
import org.keycloak.services.resources.admin.fgap.UserPermissionEvaluator;
import org.keycloak.urls.UrlType;
import org.keycloak.util.JsonSerialization;

import java.io.IOException;
import java.util.Map;

/**
 * POST /realms/{realm}/augustin-invite/link
 *   { user_id, client_id, redirect_uri, expires_in?, update_password? }
 *   → { link, expires_in, update_password }
 *
 * The link signs that user in when opened in a browser, so the caller needs
 * the same right as for impersonation, and the user has to be a plain
 * customer ({@link InvitePolicy}). update_password is only honoured for a
 * customer who has no password yet; the response says whether it was.
 */
public class InviteResource {

    private static final int DEFAULT_TTL = 120;
    private static final int MIN_TTL = 30;
    private static final int MAX_TTL = 600;

    private final KeycloakSession session;

    public InviteResource(KeycloakSession session) {
        this.session = session;
    }

    @POST
    @Path("link")
    @Consumes(MediaType.APPLICATION_JSON)
    @Produces(MediaType.APPLICATION_JSON)
    public Response link(String body) throws IOException {
        RealmModel realm = session.getContext().getRealm();

        AuthResult auth = new AppAuthManager.BearerTokenAuthenticator(session).authenticate();
        if (auth == null) {
            throw new NotAuthorizedException("Bearer");
        }

        JsonNode req;
        try {
            req = JsonSerialization.readValue(body == null || body.isBlank() ? "{}" : body, JsonNode.class);
        } catch (IOException e) {
            return error(Response.Status.BAD_REQUEST, "invalid JSON");
        }

        AdminAuth adminAuth = new AdminAuth(realm, auth.getToken(), auth.getUser(), auth.getClient());
        UserPermissionEvaluator permissions = AdminPermissions.evaluator(session, realm, adminAuth).users();

        UserModel user = session.users().getUserById(realm, req.path("user_id").asText(""));
        if (user == null) {
            if (!permissions.canImpersonate()) {
                throw new ForbiddenException();
            }
            return error(Response.Status.NOT_FOUND, "user not found");
        }
        permissions.requireImpersonate(user);

        if (!user.isEnabled()) {
            return error(Response.Status.BAD_REQUEST, "user disabled");
        }
        if (!InvitePolicy.isInvitable(realm, user)) {
            return error(Response.Status.FORBIDDEN, "invite links are only issued for customers");
        }
        ClientModel client = realm.getClientByClientId(req.path("client_id").asText(""));
        if (client == null || !client.isEnabled()) {
            return error(Response.Status.BAD_REQUEST, "unknown client_id");
        }
        String redirectUri = RedirectUtils.verifyRedirectUri(session, req.path("redirect_uri").asText(""), client);
        if (redirectUri == null) {
            return error(Response.Status.BAD_REQUEST, "redirect_uri not allowed for this client");
        }
        int ttl = Math.max(MIN_TTL, Math.min(MAX_TTL, req.path("expires_in").asInt(DEFAULT_TTL)));

        boolean updatePassword = req.path("update_password").asBoolean(false)
            && InvitePolicy.maySetPassword(realm, user);

        InviteActionToken token = new InviteActionToken(
            user.getId(), Time.currentTime() + ttl, client.getClientId(), redirectUri, updatePassword);

        // Frontend URL: the caller may reach Keycloak on an internal host,
        // the reader's browser must not
        KeycloakUriInfo uri = session.getContext().getUri(UrlType.FRONTEND);
        String link = LoginActionsService.actionTokenProcessor(uri)
            .queryParam("key", token.serialize(session, realm, uri))
            .queryParam("client_id", client.getClientId())
            .build(realm.getName())
            .toString();

        return Response.ok(JsonSerialization.writeValueAsString(Map.of("link", link, "expires_in", ttl, "update_password", updatePassword))).build();
    }

    private static Response error(Response.Status status, String message) throws IOException {
        return Response.status(status)
            .type(MediaType.APPLICATION_JSON)
            .entity(JsonSerialization.writeValueAsString(Map.of("error", message)))
            .build();
    }
}
