#!/bin/sh
# theboringoffice — TheBoringMajdoor author/committer identity. SOURCE this:
#
#   . scripts/majdoor-env.sh
#
# A git hook canNOT set authorship: git runs hooks as child processes, so
# anything a hook exports dies with the child and never reaches the git
# process that writes the commit. Authorship therefore travels as env — the
# office exports these four vars into every shell it spawns with
# THEBORINGOFFICE_AUTO_COMMIT=true; hand-rolled flows source this file for
# the same effect. When the flag is not exactly "true", nothing is exported.

if [ "${THEBORINGOFFICE_AUTO_COMMIT:-}" = "true" ]; then
    GIT_AUTHOR_NAME="TheBoringMajdoor"
    GIT_AUTHOR_EMAIL="themajdoor@theboring.name"
    GIT_COMMITTER_NAME="TheBoringMajdoor"
    GIT_COMMITTER_EMAIL="themajdoor@theboring.name"
    export GIT_AUTHOR_NAME GIT_AUTHOR_EMAIL GIT_COMMITTER_NAME GIT_COMMITTER_EMAIL
fi

# Executed directly (`sh scripts/majdoor-env.sh`)? The exports above died with
# this subprocess — the calling shell never saw them. `(return 0)` succeeds
# only inside a sourced script, so sourcing stays silent.
if ! (return 0 2>/dev/null); then
    printf '%s\n' "majdoor-env: source me instead — . scripts/majdoor-env.sh (direct execution exports into a subprocess your shell never sees)"
fi
