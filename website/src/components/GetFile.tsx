import { useEffect, useRef, useState } from "react";
import { Upload, FolderOpen, X } from "lucide-react";
import { post } from "@/utils/api";

declare module "react" {
  interface InputHTMLAttributes<T> extends HTMLAttributes<T> {
    directory?: string;
    webkitdirectory?: string;
  }
}

type file = {
  name: string;
  type: "File";
  content: File;
};

type Folder = {
  name: string;
  type: "Folder";
  Children: Node[];
};

type Node = file | Folder;

const GetFile = () => {
  const directoryRef = useRef<HTMLInputElement>(null);
  const [selectedFiles, setSelectedFiles] = useState<Node>();
  const [isDragging, setIsDragging] = useState(false);

  useEffect(() => {
    if (directoryRef.current) {
      directoryRef.current.setAttribute("webkitdirectory", "");
      directoryRef.current.setAttribute("directory", "");
    }
  }, []);

  function createSubDirectories(
    path: string[],
    currentParent: Folder,
    file: File,
  ) {
    for (let i = 0; i < path.length; i++) {
      const existingNode = currentParent.Children.find(
        (node) => node.name == path[i],
      );

      if (existingNode && existingNode.type == "Folder") {
        currentParent = existingNode;
        continue;
      }
      const newParentNode: Folder = {
        name: path[i],
        type: "Folder",
        Children: [],
      };
      currentParent.Children.push(newParentNode);
      currentParent = newParentNode;
    }
    currentParent.Children.push({
      name: file.name,
      type: "File",
      content: file,
    });
    return;
  }

  function appendToFormData(
    node: Node,
    formData: FormData,
    currentPath: string = "",
  ) {
    if (node.type === "File") {
      const fullPath = currentPath + node.name;

      formData.append(`${fullPath}`, node.content);
    }

    if (node.type === "Folder") {
      const newPath = currentPath + node.name + "/";

      for (const child of node.Children) {
        appendToFormData(child, formData, newPath);
      }
    }
  }

  function updateSelectedFile(
    parentNode: Folder,
    e: React.ChangeEvent<HTMLInputElement>,
  ) {
    let length = e.target?.files?.length;
    let fileList = e.target.files;
    if (!fileList) {
      return null;
    }

    for (let i = 0; i < length!; i++) {
      let path = fileList[i].webkitRelativePath.split("/");
      let name = path.pop();
      if (!name) {
        break;
      } // remove current file
      path.shift(); // remove already created parent

      if (path.length == 0) {
        let currentNode: file = {
          name: name,
          type: "File",
          content: fileList[i],
        };
        parentNode.Children.push(currentNode);
      } else {
        createSubDirectories(path, parentNode, fileList[i]);
      }
    }
  }

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      const files = Array.from(e.target.files);
      const folderName =
        files[0].webkitRelativePath.split("/")[0] || "Selected folder";
      let parentNode: Node = { name: folderName, type: "Folder", Children: [] };
      updateSelectedFile(parentNode, e);
      setSelectedFiles(parentNode);
    }
  };

  const clearFiles = () => {
    setSelectedFiles(undefined);
    if (directoryRef.current) {
      directoryRef.current.value = "";
    }
  };

  /**  handle the folder data uplaod  */
  const handleFileuplaod = async () => {
    const formData = new FormData();
    appendToFormData(selectedFiles!, formData);
    for (let [key, value] of formData.entries()) {
      console.log(key, value);
    }
    await post("api/upload", formData);
  };

  return (
    <div className="space-y-4">
      {/* Drop zone */}
      <div
        className={`relative flex flex-col items-center justify-center rounded-xl border-2 border-dashed p-10 transition-all ${
          isDragging
            ? "border-white/50 bg-white/5"
            : "border-white/20 bg-white/5 hover:border-white/30 hover:bg-white/5"
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
          type="file"
          ref={directoryRef}
          onChange={handleFileChange}
          className="absolute inset-0 z-10 cursor-pointer opacity-0"
        />

        <div className="flex flex-col items-center gap-4 text-center">
          <div className="flex size-14 items-center justify-center rounded-full bg-white/10">
            <Upload className="size-6 text-white/60" />
          </div>
          <div>
            <p className="text-sm font-medium text-white/80">
              Click or drag a folder here to upload
            </p>
            <p className="mt-1 text-xs text-white/40">
              Select a React project root folder
            </p>
          </div>
          <div className="flex items-center gap-2">
            <FolderOpen className="size-4 text-white/30" />
            <span className="text-xs text-white/30">
              Supports React project folders
            </span>
          </div>
        </div>
      </div>

      {/* Selected folder display */}
      {selectedFiles && (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            {/* <p className="text-xs font-medium text-white/50">
              {selectedFiles?.files.length} files selected
            </p> */}
            <button
              onClick={clearFiles}
              className="flex items-center gap-1 rounded-md px-2 py-1 text-xs text-white/40 hover:bg-white/10 hover:text-white/70 transition-all"
            >
              <X className="size-3" />
              Clear
            </button>
          </div>
          <div className="flex items-center gap-3 rounded-lg border border-white/10 bg-white/5 px-4 py-3">
            <div className="flex size-8 items-center justify-center rounded-lg bg-white/10">
              <FolderOpen className="size-4 text-white/60" />
            </div>
            <div className="flex-1 min-w-0">
              <p className="truncate text-sm font-medium text-white/80">
                {selectedFiles?.name}
              </p>
              {/* <p className="text-xs text-white/40">
                {selectedFiles?.files.length} files
              </p> */}
            </div>
          </div>
        </div>
      )}

      {/* Upload button */}
      {selectedFiles && (
        <button
          onClick={handleFileuplaod}
          className="w-full rounded-xl bg-white px-4 py-3 text-sm font-semibold text-black transition-all hover:bg-white/90 active:scale-[0.99]"
        >
          Analyze Project
        </button>
      )}
    </div>
  );
};

export default GetFile;
