import React, { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { 
  Upload, 
  FolderOpen, 
  Folder, 
  X, 
  ChevronRight, 
  FileCode, 
  File as FileIcon 
} from "lucide-react";
import { post } from "@/utils/api";

// --- Types & Declarations ---

declare module "react" {
  interface InputHTMLAttributes<T> extends HTMLAttributes<T> {
    directory?: string;
    webkitdirectory?: string;
  }
}

type FileNode = {
  name: string;
  type: "File";
  content: File;
};

type FolderNode = {
  name: string;
  type: "Folder";
  Children: Node[];
};

type Node = FileNode | FolderNode;

// --- Helper Components ---

const getFileIcon = (fileName: string) => {
  const ext = fileName.split('.').pop()?.toLowerCase();
  if (['ts', 'tsx', 'js', 'jsx'].includes(ext || '')) {
    return <FileCode className="size-3.5 text-blue-400" />;
  }
  if (['json', 'md', 'css', 'html'].includes(ext || '')) {
    return <FileIcon className="size-3.5 text-orange-300" />;
  }
  return <FileIcon className="size-3.5 text-white/40" />;
};

const TreeNode = ({ node, depth = 0 }: { node: Node; depth?: number }) => {
  const [expanded, setExpanded] = useState(depth < 1); // Auto-expand root level
  const isFolder = node.type === 'Folder';

  return (
    <div className="relative">
      {/* Visual Guide Line for nesting */}
      {depth > 0 && (
        <div 
          className="absolute left-[-11px] top-0 bottom-0 w-px bg-white/5 group-hover:bg-white/20" 
          style={{ marginLeft: `${depth === 1 ? 8 : 0}px` }}
        />
      )}

      <div className="group flex flex-col">
        {isFolder ? (
          <button
            onClick={() => setExpanded(!expanded)}
            className={`
              flex w-full items-center gap-2 rounded-md px-2 py-1 text-sm transition-colors
              hover:bg-white/[0.06] focus:outline-none
              ${depth === 0 ? 'mt-1 mb-1' : ''}
            `}
          >
            <div className="flex size-4 items-center justify-center">
              <ChevronRight 
                className={`size-3 text-white/30 transition-transform duration-200 ${expanded ? 'rotate-90' : ''}`} 
              />
            </div>
            
            {expanded ? (
              <FolderOpen className="size-4 text-amber-400/70" />
            ) : (
              <Folder className="size-4 text-amber-400/70" />
            )}
            
            <span className={`truncate ${depth === 0 ? 'font-semibold text-white/90' : 'text-white/70'}`}>
              {node.name}
            </span>
          </button>
        ) : (
          <div className="flex w-full items-center gap-2 rounded-md px-2 py-1 text-sm transition-colors hover:bg-white/[0.04]">
            <div className="size-4" /> {/* Alignment Spacer */}
            {getFileIcon(node.name)}
            <span className="truncate text-white/50 group-hover:text-white/70">
              {node.name}
            </span>
          </div>
        )}

        {isFolder && expanded && (
          <div className="ml-[19px] flex flex-col">
            {node.Children.map((child, i) => (
              <TreeNode key={`${child.name}-${i}`} node={child} depth={depth + 1} />
            ))}
            {node.Children.length === 0 && (
              <div className="py-1 pl-8 text-[10px] italic text-white/20">
                Empty folder
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

// --- Main Component ---

interface GetFileProps {
  onUploadSuccess?: () => void;
}

const GetFile = ({ onUploadSuccess }: GetFileProps) => {
  const navigate = useNavigate();
  const directoryRef = useRef<HTMLInputElement>(null);
  const [selectedFiles, setSelectedFiles] = useState<Node>();
  const [isDragging, setIsDragging] = useState(false);
  const [showTree, setShowTree] = useState(false);
  const [projectName, setProjectName] = useState("");

  useEffect(() => {
    if (directoryRef.current) {
      directoryRef.current.setAttribute("webkitdirectory", "");
      directoryRef.current.setAttribute("directory", "");
    }
  }, []);

  function createSubDirectories(path: string[], currentParent: FolderNode, file: File) {
    for (let i = 0; i < path.length; i++) {
      let existingNode = currentParent.Children.find(
        (node) => node.name === path[i] && node.type === "Folder"
      ) as FolderNode | undefined;

      if (!existingNode) {
        existingNode = {
          name: path[i],
          type: "Folder",
          Children: [],
        };
        currentParent.Children.push(existingNode);
      }
      currentParent = existingNode;
    }
    
    currentParent.Children.push({
      name: file.name,
      type: "File",
      content: file,
    });
  }

  function updateSelectedFile(parentNode: FolderNode, fileList : FileList) {
    if (!fileList) return;

    for (let i = 0; i < fileList.length; i++) {
      const file = fileList[i];
      const path = file.webkitRelativePath.split("/");
      path.shift(); // Remove the root folder name (already handled by parentNode)
      const fileName = path.pop();

      if (!fileName) continue;

      if (path.length === 0) {
        parentNode.Children.push({
          name: fileName,
          type: "File",
          content: file,
        });
      } else {
        createSubDirectories(path, parentNode, file);
      }
    }
  }

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      const folderName = e.target.files[0].webkitRelativePath.split("/")[0] || "Project Root";
      const rootNode: FolderNode = { name: folderName, type: "Folder", Children: [] };
      updateSelectedFile(rootNode, e.target.files);
      setSelectedFiles(rootNode);
    }
  };

  const clearFiles = () => {
    setSelectedFiles(undefined);
    setShowTree(false);
    setProjectName("");
    if (directoryRef.current) directoryRef.current.value = "";
  };

  const handleFileUpload = async () => {
    if (!selectedFiles) return;
    if (!projectName.trim()) {
      alert("Please enter a project name.");
      return;
    }

    const formData = new FormData();
    formData.append("projectName", projectName.trim());
    const appendToFormData = (node: Node, currentPath: string = "") => {
      if (node.type === "File") {
        formData.append(`${currentPath}${node.name}`, node.content);
      } else {
        const newPath = `${currentPath}${node.name}/`;
        node.Children.forEach(child => appendToFormData(child, newPath));
      }
    };

    appendToFormData(selectedFiles);
    
    try {
      const res = await post<{ deployment_id: number }>("api/upload", formData);
      onUploadSuccess?.();
      navigate(`/project/${res.deployment_id}`);
    } catch (error) {
      console.error("Upload failed", error);
    }
  };

  return (
    <div className="mx-auto max-w-2xl space-y-4 p-6">
      {/* Drop Zone */}
      {!selectedFiles && (
        <div
          className={`relative flex flex-col items-center justify-center rounded-xl border-2 border-dashed p-12 transition-all ${
            isDragging
              ? "border-white/50 bg-white/10"
              : "border-white/10 bg-white/5 hover:border-white/20"
          }`}
          onDragOver={(e) => { e.preventDefault(); setIsDragging(true); }}
          onDragLeave={() => setIsDragging(false)}
          onDrop={(e) => { e.preventDefault(); setIsDragging(false); }}
        >
          <input
            type="file"
            webkitdirectory=""
            ref={directoryRef}
            onChange={handleFileChange}
            className="absolute inset-0 z-10 cursor-pointer opacity-0"
          />
          <div className="flex flex-col items-center gap-4 text-center">
            <div className="flex size-14 items-center justify-center rounded-full bg-white/10 text-white/60">
              <Upload className="size-6" />
            </div>
            <div>
              <p className="text-sm font-medium text-white/80">Select project folder</p>
              <p className="mt-1 text-xs text-white/40">Drag and drop or click to browse</p>
            </div>
          </div>
        </div>
      )}

      {/* Preview Section */}
      {selectedFiles && (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="flex size-10 items-center justify-center rounded-lg bg-amber-400/10">
                <FolderOpen className="size-5 text-amber-400/80" />
              </div>
              <div>
                <p className="text-sm font-semibold text-white/90">{selectedFiles.name}</p>
                <p className="text-[10px] uppercase tracking-wider text-white/30">Ready to process</p>
              </div>
            </div>
            <button
              onClick={clearFiles}
              className="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium text-white/40 hover:bg-white/10 hover:text-red-400 transition-all"
            >
              <X className="size-3" />
              Reset
            </button>
          </div>

          {/* Project Name Input */}
          <div className="space-y-1.5 text-left">
            <label className="text-xs font-medium text-white/50">Project Name</label>
            <input
              type="text"
              value={projectName}
              onChange={(e) => setProjectName(e.target.value)}
              placeholder="Enter a project name..."
              className="w-full rounded-lg border border-white/10 bg-white/5 px-3.5 py-2 text-sm text-white placeholder-white/20 focus:border-indigo-500 focus:outline-none"
            />
          </div>

          {!showTree ? (
            <button
              onClick={() => setShowTree(true)}
              className="w-full rounded-xl bg-white px-4 py-3 text-sm font-bold text-black transition-all hover:bg-white/90 active:scale-[0.98]"
            >
              Examine Folder Structure
            </button>
          ) : (
            <div className="space-y-4">
              <div className="max-h-[400px] overflow-y-auto rounded-xl border border-white/10 bg-[#0A0A0A] p-4 shadow-2xl">
                <TreeNode node={selectedFiles} />
              </div>
              
              <button
                onClick={handleFileUpload}
                className="w-full rounded-xl bg-indigo-500 px-4 py-3 text-sm font-bold text-white transition-all hover:bg-indigo-400 active:scale-[0.98]"
              >
                Upload Project
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default GetFile;