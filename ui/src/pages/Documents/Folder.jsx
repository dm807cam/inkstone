import { useState } from "react";
import { Button, InputGroup, Form } from "react-bootstrap";
import Modal from 'react-bootstrap/Modal';
import { BsFillGridFill } from "react-icons/bs";
import { FaList } from "react-icons/fa";
import { ToggleButton, ToggleButtonGroup } from "react-bootstrap";

import apiservice from "../../services/api.service"
import styles from "./Documents.module.scss"

import Upload from "./Upload"
import FileList from "./FileList";
import NameTag from "../../components/NameTag"
import { toast } from "react-toastify";

export default function Folder({ selection, onSelect, onUpdate }) {
  const [listStyle, setListStyle] = useState("list");
  const [folderName, setFolderName] = useState("");
  const [showCreateFileModal, setShowCreateFolder] = useState(false);
  const [selectedIds, setSelectedIds] = useState([]);

  const folder = selection
  const isTrash = folder?.id === "trash";

  const onCreateFolderClick = async () => {
    const res = await apiservice.createFolder({ name: folderName, parentId: selection.id });
    console.log("created folder with id", res.ID);
    setFolderName("");
    setShowCreateFolder(false);

    onUpdate();
  }

  const onDeleteClick = async () => {
    if (selectedIds.length === 0) return;
    if (!window.confirm(`Are you sure you want to delete the selected item(s)?`)) return;
    for (const id of selectedIds) {
      const file = folder.children.find(f => f.id === id);
      const name = file?.data?.name || id;
      try {
        await apiservice.deleteDocument(id);
        toast.success(`Deleted ${name}`);
      } catch (e) {
        toast.error(`Failed to delete ${name}`);
      }
    }
    setSelectedIds([]);
    onUpdate();
  }

  // Restore trashed items by moving them back to the root. The update endpoint
  // replaces name and parent together, so the current name is re-sent unchanged
  // and the parent is set to "" (root).
  const onRestoreClick = async () => {
    if (selectedIds.length === 0) return;
    for (const id of selectedIds) {
      const item = folder.children?.find((f) => f.id === id);
      const name = item?.data?.name || id;
      try {
        await apiservice.updateDocument({ documentId: id, name, parentId: "" });
        toast.success(`Restored ${name}`);
      } catch (e) {
        toast.error(`Failed to restore ${name}`);
      }
    }
    setSelectedIds([]);
    onUpdate();
  }

  const fileUploaded = () => {
    onUpdate();
  }

  const handleSelectItem = (id) => {
    setSelectedIds((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : [...prev, id]
    );
  };

  // this should generally not happen, but just in case
  if (!folder) {
    return "nothing selected"
  }
  return (
    <>
      <div className={styles.breadcrumbBar}>
        { folder && <NameTag node={folder} onSelect={onSelect} /> }
      </div>

      <div className={`${styles.toolbar} ${styles.filedivider}`}>
        {!isTrash && <Button size="sm" variant="outline" onClick={() => setShowCreateFolder(true)}>Create Folder</Button>}
        <div className={styles.stretch}></div>
        {isTrash && <Button size="sm" variant="success" onClick={onRestoreClick} disabled={selectedIds.length === 0}>Restore</Button>}
        <Button size="sm" variant="danger" onClick={onDeleteClick} disabled={selectedIds.length === 0}>{isTrash ? "Delete forever" : "Delete"}</Button>
        <ToggleButtonGroup value={listStyle} onChange={(v) => setListStyle(v)} name="listStyle">
          <ToggleButton id="grid" name="grid" size="sm" value="grid" variant="outline">
            <BsFillGridFill />
          </ToggleButton>
          <ToggleButton id="list" name="list" size="sm" value="list" variant="outline">
            <FaList />
          </ToggleButton>
        </ToggleButtonGroup>
      </div>

      {!isTrash && <Upload filesUploaded={fileUploaded} uploadFolder={selection.id}></Upload>}
      <FileList
        listStyle={listStyle}
        files={folder.children}
        onSelect={onSelect}
        selectedIds={selectedIds}
        onSelectItem={handleSelectItem}
      />

      <Modal show={showCreateFileModal} onHide={() => setShowCreateFolder(false)}>
        <Modal.Header closeButton>
          Create a new folder
        </Modal.Header>

        <Modal.Body>
          <InputGroup className="mb-3">
            <Form.Control autoFocus={true} type="text" value={folderName} onChange={(e) => setFolderName(e.currentTarget.value)} />

            <Button variant="primary" onClick={onCreateFolderClick}>Create</Button>

          </InputGroup>
        </Modal.Body>
      </Modal>
    </>
  );
}
