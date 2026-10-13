package at.augustin.keycloak.invite;

import jakarta.ws.rs.core.Response;
import org.keycloak.TokenVerifier.Predicate;
import org.keycloak.authentication.actiontoken.AbstractActionTokenHandler;
import org.keycloak.authentication.actiontoken.ActionTokenContext;
import org.keycloak.authentication.actiontoken.TokenUtils;
import org.keycloak.events.Errors;
import org.keycloak.events.EventType;
import org.keycloak.models.KeycloakSession;
import org.keycloak.models.UserModel;
import org.keycloak.protocol.oidc.OIDCLoginProtocol;
import org.keycloak.protocol.oidc.utils.RedirectUtils;
import org.keycloak.services.managers.AuthenticationManager;
import org.keycloak.services.messages.Messages;
import org.keycloak.sessions.AuthenticationSessionModel;

/**
 * Opened in the browser: the invited reader is authenticated without a
 * password, sets one if the invite asks for it and none exists yet
 * (Keycloak's own «update password» page) and returns to the app with a
 * normal SSO session.
 *
 * Modelled on Keycloak's ExecuteActionsActionTokenHandler, which ends after
 * the required actions without signing the user in.
 */
public class InviteActionTokenHandler extends AbstractActionTokenHandler<InviteActionToken> {

    public InviteActionTokenHandler() {
        super(InviteActionToken.TOKEN_TYPE, InviteActionToken.class, Messages.INVALID_CODE,
              EventType.EXECUTE_ACTION_TOKEN, Errors.NOT_ALLOWED);
    }

    @Override
    public Predicate<? super InviteActionToken>[] getVerifiers(ActionTokenContext<InviteActionToken> ctx) {
        return TokenUtils.predicates(
            TokenUtils.checkThat(
                (InviteActionToken t) -> InvitePolicy.isInvitable(
                    ctx.getRealm(), ctx.getAuthenticationSession().getAuthenticatedUser()),
                Errors.NOT_ALLOWED, Messages.INVALID_USER),
            TokenUtils.checkThat(
                (InviteActionToken t) -> RedirectUtils.verifyRedirectUri(
                    ctx.getSession(), t.getRedirectUri(), ctx.getAuthenticationSession().getClient()) != null,
                Errors.INVALID_REDIRECT_URI, Messages.INVALID_REDIRECT_URI)
        );
    }

    /** Unlike the base class: no END_AFTER_REQUIRED_ACTIONS – the flow ends in a login. */
    @Override
    public AuthenticationSessionModel startFreshAuthenticationSession(InviteActionToken token,
                                                                      ActionTokenContext<InviteActionToken> ctx) {
        return ctx.createAuthenticationSessionForClient(token.getIssuedFor());
    }

    @Override
    public Response handleToken(InviteActionToken token, ActionTokenContext<InviteActionToken> ctx) {
        KeycloakSession session = ctx.getSession();
        AuthenticationSessionModel authSession = ctx.getAuthenticationSession();

        String redirectUri = RedirectUtils.verifyRedirectUri(session, token.getRedirectUri(), authSession.getClient());
        authSession.setRedirectUri(redirectUri);
        authSession.setClientNote(OIDCLoginProtocol.REDIRECT_URI_PARAM, redirectUri);
        authSession.setClientNote(OIDCLoginProtocol.RESPONSE_TYPE_PARAM, "code");
        authSession.setClientNote(OIDCLoginProtocol.SCOPE_PARAM, "openid");
        AuthenticationManager.setClientScopesInSession(session, authSession);

        // The invite reached the reader by e-mail
        UserModel user = authSession.getAuthenticatedUser();
        user.setEmailVerified(true);

        if (token.isUpdatePassword() && InvitePolicy.maySetPassword(ctx.getRealm(), user)) {
            authSession.addRequiredAction(UserModel.RequiredAction.UPDATE_PASSWORD);
        }

        String next = AuthenticationManager.nextRequiredAction(session, authSession, ctx.getRequest(), ctx.getEvent());
        return AuthenticationManager.redirectToRequiredActions(session, ctx.getRealm(), authSession, ctx.getUriInfo(), next);
    }

    @Override
    public boolean canUseTokenRepeatedly(InviteActionToken token, ActionTokenContext<InviteActionToken> ctx) {
        return false;
    }
}
