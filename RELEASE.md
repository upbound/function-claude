## Release Process

This is a slimmed-down version of the release process described [here](https://github.com/crossplane/release).

1. **feature freeze**: Merge all completed features into main development branch
   of all repos to begin "feature freeze" period.
1. **branch repo**: Create a new release branch using the GitHub UI for the
   repo (e.g. `release-0.25`).
1. **tag release**: Run the `Tag` action on the _release branch_ with the
   desired version (e.g. `v0.25.0`).
1. **build/publish**: Run the `CI` action on the tag.
1. **tag next pre-release**: Run the `tag` action on the main development branch
   with the `-0.rc.0` for the next release (e.g. `v0.26.0-0.rc.0`). (**NOTE**:
   we added the `-0.` prefix to allow correctly sorting release candidates)
1. **verify**: Verify all artifacts have been published successfully, perform
   sanity testing.
   - Check in Upbound Marketplace that the new versions are present.
1. **release notes**:
   - Open the new release tag in https://github.com/upbound/function-claude/tags and click "Create
     release from tag".
   - "Generate release notes" from previous release ("auto" might not work).
   - Make sure the release notes are complete, presize and well formatted.
   - Publish the well authored Github release.
1. **announce**: Announce the release on Twitter, Slack, etc.
   - Crossplane Slack #Upbound: https://crossplane.slack.com/archives/C01TRKD4623
   - TODO: where else?