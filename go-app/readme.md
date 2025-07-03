# site-usage statistics


This is the repo for the statistics and status backend service
for the https://status.arc42.org website.

The documentation is contained in the [status-repository](https://github.com/arc42/status.arc42.org-site) repository.

## Environment Variables

This application requires the following environment variables to be set:

- `GITHUB_API_KEY`: GitHub API token for accessing repository statistics (issues, bugs)
- `PLAUSIBLE_API_KEY`: Plausible.io API key for web analytics 
- `TURSO_AUTH_TOKEN`: Turso database authentication token
- `LOGLEVEL`: Log level (debug, info, warn, error)

## Troubleshooting

### GitHub Issue Counts Show Zero

If all GitHub issue counts appear as 0 in the statistics table:

1. **Check API Key**: Ensure `GITHUB_API_KEY` environment variable is set with a valid GitHub token
2. **Check Logs**: Look for error messages indicating API failures
3. **Network Issues**: Verify GitHub API access is not blocked by firewalls or proxies
4. **Rate Limits**: Check if GitHub API rate limits have been exceeded

The application will gracefully handle GitHub API failures and preserve existing values instead of overwriting with zeros.

