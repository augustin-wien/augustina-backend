package at.augustin.keycloak.invite;

import org.keycloak.Config;
import org.keycloak.models.KeycloakSession;
import org.keycloak.models.KeycloakSessionFactory;
import org.keycloak.services.resource.RealmResourceProvider;
import org.keycloak.services.resource.RealmResourceProviderFactory;

/** Mounts {@link InviteResource} at /realms/{realm}/augustin-invite. */
public class InviteResourceProviderFactory implements RealmResourceProviderFactory, RealmResourceProvider {

    public static final String ID = "augustin-invite";

    private KeycloakSession session;

    public InviteResourceProviderFactory() {
    }

    private InviteResourceProviderFactory(KeycloakSession session) {
        this.session = session;
    }

    @Override
    public RealmResourceProvider create(KeycloakSession session) {
        return new InviteResourceProviderFactory(session);
    }

    @Override
    public Object getResource() {
        return new InviteResource(session);
    }

    @Override
    public String getId() {
        return ID;
    }

    @Override
    public void init(Config.Scope config) {
    }

    @Override
    public void postInit(KeycloakSessionFactory factory) {
    }

    @Override
    public void close() {
    }
}
