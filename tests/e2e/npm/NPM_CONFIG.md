# NPM Configuration for Artifusion

## Correct .npmrc Format

The NPM test script uses the following `.npmrc` configuration format:

```ini
registry=http://artifacts.lvh.me/npm/
//artifacts.lvh.me/npm/:_authToken=ghp_your_github_token
//artifacts.lvh.me/npm/:username=your-github-username
//artifacts.lvh.me/npm/:email=your-email@example.com
```

## Authentication Methods

### Method 1: Using _authToken (Recommended)

The `_authToken` is the modern way to authenticate with npm registries. When set for a specific registry scope, npm automatically uses this token for all requests to that registry.

```ini
//artifacts.lvh.me/npm/:_authToken=${GITHUB_TOKEN}
```

**Advantages**:
- Works with all modern npm versions
- Automatically used for the scoped registry
- No deprecated configuration warnings

### Method 2: Using npm login (Alternative)

You can also authenticate interactively:

```bash
npm login --registry=http://artifacts.lvh.me/npm/

# Enter credentials when prompted:
# Username: your-github-username
# Password: ghp_your_github_token
# Email: your-email@example.com
```

This creates the same `_authToken` entry in your `.npmrc`.

## What Changed?

### ❌ Old (Deprecated)
```ini
//artifacts.lvh.me/npm/:always-auth=true
```

**Issue**:
- `always-auth` is deprecated in npm 7+
- Produces warning: `Unknown user config "always-auth"`
- Will stop working in future npm versions

### ✅ New (Correct)
```ini
# No always-auth needed!
# The _authToken is automatically used
//artifacts.lvh.me/npm/:_authToken=${GITHUB_TOKEN}
```

**Why it works**:
- Modern npm automatically uses `_authToken` when configured
- No need for explicit `always-auth` flag
- Cleaner, more maintainable configuration

## Full Example

### For Testing (Temporary .npmrc)
```bash
# Create temporary .npmrc
cat > /tmp/.npmrc << EOF
registry=http://artifacts.lvh.me/npm/
//artifacts.lvh.me/npm/:_authToken=${GITHUB_TOKEN}
//artifacts.lvh.me/npm/:username=${GITHUB_USERNAME}
//artifacts.lvh.me/npm/:email=${GITHUB_EMAIL}
EOF

# Use it
npm install --userconfig=/tmp/.npmrc
```

### For Development (~/.npmrc)
```bash
# Add to your ~/.npmrc
npm config set registry http://artifacts.lvh.me/npm/
npm config set //artifacts.lvh.me/npm/:_authToken ${GITHUB_TOKEN}
npm config set //artifacts.lvh.me/npm/:username ${GITHUB_USERNAME}
npm config set //artifacts.lvh.me/npm/:email ${GITHUB_EMAIL}
```

### For CI/CD (.npmrc in project)
```ini
# Project .npmrc (committed to repo)
registry=http://artifacts.lvh.me/npm/

# .npmrc (created during CI from secrets)
//artifacts.lvh.me/npm/:_authToken=${NPM_AUTH_TOKEN}
```

## Scoped Registries

If you only want certain packages to use Artifusion (not all packages):

```ini
# Keep public registry as default
registry=https://registry.npmjs.org/

# Use Artifusion for specific scope
@myorg:registry=http://artifacts.lvh.me/npm/
//artifacts.lvh.me/npm/:_authToken=${GITHUB_TOKEN}
```

Now only `@myorg/*` packages will use Artifusion, others use npmjs.org.

## Troubleshooting

### Warning: "Unknown user config"
**Symptom**:
```
npm warn Unknown user config "always-auth"
```

**Solution**:
Remove `always-auth` from your `.npmrc`. It's no longer needed.

### 401 Unauthorized
**Symptom**:
```
npm ERR! code E401
npm ERR! 401 Unauthorized
```

**Solutions**:
1. Check token is valid:
   ```bash
   curl -H "Authorization: token $GITHUB_TOKEN" https://api.github.com/user
   ```

2. Verify token has correct scopes: `read:packages`, `read:user`

3. Check `.npmrc` format (no typos, correct registry URL)

4. Test authentication:
   ```bash
   npm whoami --registry=http://artifacts.lvh.me/npm/
   ```

### Cannot resolve packages
**Symptom**:
```
npm ERR! 404 Not Found
```

**Solutions**:
1. Check Artifusion is running:
   ```bash
   curl http://artifacts.lvh.me/health
   ```

2. Verify registry URL in `.npmrc`

3. Check backend (Verdaccio) is healthy:
   ```bash
   kubectl get pods -n artifusion
   kubectl logs -n artifusion artifusion-verdaccio-0
   ```

## npm Version Compatibility

| npm Version | _authToken | always-auth | Notes |
|-------------|------------|-------------|-------|
| npm 6.x     | ✅ Yes     | ⚠️ Optional | `always-auth` works but not needed |
| npm 7.x     | ✅ Yes     | ⚠️ Deprecated | Produces warning |
| npm 8.x+    | ✅ Yes     | ❌ Removed  | Will error if used |

**Recommendation**: Always use `_authToken` without `always-auth` for maximum compatibility.

## Security Best Practices

1. **Never commit tokens** to git:
   ```bash
   # Add to .gitignore
   echo ".npmrc" >> .gitignore
   ```

2. **Use environment variables**:
   ```bash
   # In shell
   export GITHUB_TOKEN=ghp_xxx

   # In .npmrc
   //artifacts.lvh.me/npm/:_authToken=${GITHUB_TOKEN}
   ```

3. **Rotate tokens regularly**:
   - Generate new token every 90 days
   - Revoke old tokens immediately
   - Use fine-grained tokens when possible

4. **Limit token scopes**:
   - Only grant `read:packages` for consuming
   - Add `write:packages` only if publishing
   - Never grant more than needed

## References

- npm config documentation: https://docs.npmjs.com/cli/v9/configuring-npm/npmrc
- npm authentication: https://docs.npmjs.com/about-authentication-tokens
- GitHub PAT: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token
