This directory is an example of default models directories structure:

models/
       bin/
            : default location of model executables and model.sqlite database files.
              To change default location use: -oms.ModelDir
       log/
            : default location of model run log files
	      To change it use: -oms.ModelLogDir
       doc/
            : default location of model documentation
	      To change it use: -oms.ModelDocDir

      home/io/
              : default location of user data, model downloads and model uploads.
                To change default location use:     -oms.HomeDir
            download
                     : optional folder, to support download of model parameters and output data.
                       To enable downloads use: -oms.AllowDownload
            upload
                     : optional folder, to support download of model parameters and output data.
                       To enable uploads use: -oms.AllowUpload
		       
Example:

cd openmpp-root-dir
bin/oms -oms.HomeDir models/homeAnyWhere -oms.AllowDownload -oms.AllowUpload
