## update AWE version

Create tag:
```bash
git tag -a v0.9.70-dev -m v0.9.70-dev
```

Push change to repo:
```bash
git push origin --tag v0.9.70-dev
```

Make tag a release (for non dev tags):
1. go to https://github.com/MG-RAST/AWE/tags
2. click on '...' next to tag
3. edit release with added notes and / or compiled binary