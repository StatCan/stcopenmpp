"""
Repository cloning functionality for STCOpenMPP testing.
"""

import os
import shutil
import glob
from pathlib import Path
from git import Repo, rmtree
from urllib.parse import urlparse
import click


def clone_repo(git_url, git_username=None, git_password=None, git_commit=None, model_sln=None):
    """
    Clone a Git repository and find the model solution file.
    
    If you give it credentials, it'll use them. If you want a specific commit, 
    it'll check that out. If you tell it what model file to look for, it'll find it.
    """
    click.echo(f"   Cloning repo {git_url}. Hang tight ...")
    
    # Update git_url if login credentials are set
    if git_username and git_password:
        parsed_url = urlparse(git_url)
        git_url = f"{parsed_url.scheme}://{git_username}:{git_password}@{parsed_url.netloc}{parsed_url.path}"
    
    # Get local repo path
    repo_name = Path(model_sln).stem.replace('-ompp', '')  # Name cloned repo to match solution file.
    local_path = Path.cwd() / repo_name
    
    # Remove local repo if it exists
    if local_path.exists():
        rmtree(local_path)
    
    try:
        # Clone repo
        repo = Repo.clone_from(git_url, local_path)
        click.echo(f"   Repository cloning successful.")
        
        # Checkout commit hash if set
        if git_commit and git_commit.lower() != "latest":
            repo.git.checkout(git_commit, force=True)
            click.echo(f"   Checked out commit {git_commit}")
        
        # Find model solution file
        if model_sln:
            model_files = list(local_path.glob(f"**/{model_sln}"))
            
            if model_files:
                model_path = str(model_files[0])
                click.echo(f"   Found model solution: {model_path}")
                return model_path
            else:
                raise FileNotFoundError(f"  Can't find {model_sln} anywhere in the repo")
        
        return str(local_path)
        
    except Exception as e:
        click.echo(f"Something went wrong: {str(e)}")
        raise 