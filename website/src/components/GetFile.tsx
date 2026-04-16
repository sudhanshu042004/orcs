import { useEffect, useRef, useState } from 'react';
import { Upload, FolderOpen, X } from 'lucide-react';

declare module 'react' {
  interface InputHTMLAttributes<T> extends HTMLAttributes<T> {
    directory?: string;
    webkitdirectory?: string;
  }
}

interface SelectedFile {
  name: string;
  path: string;
  files: File[];
}

const GetFile = () => {
  const directoryRef = useRef<HTMLInputElement>(null);
  const [selectedFiles, setSelectedFiles] = useState<SelectedFile[]>([]);
  const [isDragging, setIsDragging] = useState(false);

  useEffect(() => {
    if (directoryRef.current) {
      directoryRef.current.setAttribute('webkitdirectory', '');
      directoryRef.current.setAttribute('directory', '');
    }
  }, []);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      const files = Array.from(e.target.files);
      const folderName = files[0].webkitRelativePath.split('/')[0] || 'Selected folder';
      setSelectedFiles([{ name: folderName, path: '', files }]);
    }
  };

  const clearFiles = () => {
    setSelectedFiles([]);
    if (directoryRef.current) {
      directoryRef.current.value = '';
    }
  };

  return (
    <div className='space-y-4'>
      {/* Drop zone */}
      <div
        className={`relative flex flex-col items-center justify-center rounded-xl border-2 border-dashed p-10 transition-all ${
          isDragging
            ? 'border-white/50 bg-white/5'
            : 'border-white/20 bg-white/5 hover:border-white/30 hover:bg-white/5'
        }`}
        onDragOver={(e) => {
          e.preventDefault();
          setIsDragging(true);
        }}
        onDragLeave={() => setIsDragging(false)}
        onDrop={(e) => {
          e.preventDefault();
          setIsDragging(false);
        }}
      >
        <input
          type='file'
          ref={directoryRef}
          onChange={handleFileChange}
          className='absolute inset-0 z-10 cursor-pointer opacity-0'
        />

        <div className='flex flex-col items-center gap-4 text-center'>
          <div className='flex size-14 items-center justify-center rounded-full bg-white/10'>
            <Upload className='size-6 text-white/60' />
          </div>
          <div>
            <p className='text-sm font-medium text-white/80'>
              Click or drag a folder here to upload
            </p>
            <p className='mt-1 text-xs text-white/40'>
              Select a React project root folder
            </p>
          </div>
          <div className='flex items-center gap-2'>
            <FolderOpen className='size-4 text-white/30' />
            <span className='text-xs text-white/30'>
              Supports React project folders
            </span>
          </div>
        </div>
      </div>

      {/* Selected folder display */}
      {selectedFiles.length > 0 && (
        <div className='space-y-2'>
          <div className='flex items-center justify-between'>
            <p className='text-xs font-medium text-white/50'>
              {selectedFiles[0].files.length} files selected
            </p>
            <button
              onClick={clearFiles}
              className='flex items-center gap-1 rounded-md px-2 py-1 text-xs text-white/40 hover:bg-white/10 hover:text-white/70 transition-all'
            >
              <X className='size-3' />
              Clear
            </button>
          </div>
          <div className='flex items-center gap-3 rounded-lg border border-white/10 bg-white/5 px-4 py-3'>
            <div className='flex size-8 items-center justify-center rounded-lg bg-white/10'>
              <FolderOpen className='size-4 text-white/60' />
            </div>
            <div className='flex-1 min-w-0'>
              <p className='truncate text-sm font-medium text-white/80'>
                {selectedFiles[0].name}
              </p>
              <p className='text-xs text-white/40'>
                {selectedFiles[0].files.length} files
              </p>
            </div>
          </div>
        </div>
      )}

      {/* Upload button */}
      {selectedFiles.length > 0 && (
        <button className='w-full rounded-xl bg-white px-4 py-3 text-sm font-semibold text-black transition-all hover:bg-white/90 active:scale-[0.99]'>
          Analyze Project
        </button>
      )}
    </div>
  );
};

export default GetFile;