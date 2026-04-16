"""
Model building functionality for STCOpenMPP testing.
"""

import os
import shutil
import subprocess
import glob
from pathlib import Path
import click


def build_model(model_sln, om_root, vs_cmd_path, mode="release", bit=64):
    """
    Build STCOpenMPP models using MSBuild.
    
    Takes your model solution file and builds it for each STCOpenMPP version you give it.
    Copies all the necessary files to the right places so the models can run.
    """
    click.echo(f"   Model solution file located at: {model_sln}")
    
    if not Path(model_sln).exists():
        raise FileNotFoundError(f"Can't find the model file: {model_sln}")
    
    model_names = []
    
    for root in om_root:
        click.echo(f"   Building for STCOpenMPP at {root}...")
        
        if not Path(root).exists():
            click.echo(f"   STCOpenMPP directory doesn't exist: {root}")
            continue
        
        try:
            model_name = _build_single_model(model_sln, root, vs_cmd_path, mode, bit)
            if model_name:
                model_names.append(model_name)
                click.echo(f"       Built {model_name} for {root}")
            else:
                click.echo(f"       Build failed for {root}")
                
        except Exception as e:
            click.echo(f"       Build error for {root}: {str(e)}")
    
    return model_names


def _build_single_model(model_sln, om_root, vs_cmd_path, mode, bit):
    """Build a model for one specific STCOpenMPP installation."""
    
    model_dir = Path(model_sln).parent
    model_name = Path(model_sln).stem.replace('-ompp', '')

    click.echo(f"       Setting up environment for {model_name}")
    
    env = os.environ.copy()
    env['OM_ROOT'] = str(Path(om_root).resolve())
    env['PATH'] = f"{env['OM_ROOT']}/bin;{env['PATH']}"
    
    click.echo(f"       Running MSBuild...")
    
    try:
        # Handle Visual Studio command path properly
        if vs_cmd_path.endswith('.bat'):
            # If it's VsDevCmd.bat, set up environment first
            vs_setup_cmd = f'call "{vs_cmd_path}"'
            msbuild_cmd = f'msbuild "{model_sln}" /t:Clean,Build /p:Configuration={mode} /p:OM_ROOT={om_root} /p:Platform=x{bit} /p:RUN_SCENARIO=False /p:OPEN_MODEL_WEB_UI=False'
            full_cmd = f'{vs_setup_cmd} && {msbuild_cmd}'
        else:
            # If it's direct MSBuild.exe path
            full_cmd = f'"{vs_cmd_path}" "{model_sln}" /t:Clean,Build /p:Configuration={mode} /p:OM_ROOT={om_root} /p:Platform=x{bit} /p:RUN_SCENARIO=False /p:OPEN_MODEL_WEB_UI=False'
        
        result = subprocess.run(
            full_cmd,
            cwd=model_dir,
            env=env,
            capture_output=True,
            text=True,
            shell=True
        )
        
        if result.returncode != 0:
            click.echo(f"           MSBuild failed with return code {result.returncode}")
            if result.stdout:
                click.echo(f"           MSBuild stdout: {result.stdout}")
            if result.stderr:
                click.echo(f"           MSBuild stderr: {result.stderr}")
            
            # Check for specific error patterns
            error_output = (result.stdout or "") + (result.stderr or "")
            if "error C" in error_output or "Error(s)" in error_output:
                click.echo(f"    Compilation errors detected - build failed")
                raise RuntimeError(f"Model compilation failed with errors")
            
            return None
        
        click.echo(f"           MSBuild completed successfully")
        
        click.echo(f"       Copying model files...")
        _copy_model_files(model_dir, om_root, model_name)
        
        return model_name
        
    except Exception as e:
        click.echo(f"    Build process failed: {str(e)}")
        return None


def _copy_model_files(model_dir, om_root, model_name):
    """Copy the built model files to the STCOpenMPP installation."""
    
    # Create the model-specific directory structure that STCOpenMPP expects
    model_bin_dir = Path(om_root) / 'models' / model_name / 'ompp' / 'bin'
    model_bin_dir.mkdir(parents=True, exist_ok=True)
    
    # Also copy to the standard models/bin directory for backward compatibility
    om_models_bin = Path(om_root) / 'models' / 'bin'
    om_models_bin.mkdir(parents=True, exist_ok=True)
    
    model_files = [
        f"{model_name}.exe",
        f"{model_name}.dll", 
        f"{model_name}.pdb",
        f"{model_name}.xml",
        f"{model_name}.sqlite"
    ]
    
    for file_pattern in model_files:
        files = list(model_dir.glob(f"**/{file_pattern}"))
        
        for file_path in files:
            # Copy to model-specific directory (for start-ompp-ui.bat)
            dest_path_model = model_bin_dir / file_path.name
            # Copy to standard models/bin directory (for API service)
            dest_path_standard = om_models_bin / file_path.name
            
            try:
                shutil.copy2(file_path, dest_path_model)
                shutil.copy2(file_path, dest_path_standard)
                click.echo(f"           Copied {file_path.name} to both locations")
            except Exception as e:
                click.echo(f"           Failed to copy {file_path.name}: {str(e)}")