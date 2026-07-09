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
  const [renameName, setRenameName] = useState("");
  const [showRenameModal, setShowRenameModal] = useState(false);

  const folder = selection
  const isTrash = folder?.id === "trash";

  // The backend stores a document's parent as an id, with the empty string
  // meaning "root". The tree exposes the root as the pseudo-id "root", so map
  // it back so rename/move keeps items at the top level instead of orphaning
  // them under a non-existent "root" folder.
  const currentParentId = folder?.id === "root" ? "" : folder?.id;

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

  const selectedItem = selectedIds.length === 1
    ? folder?.children?.find((f) => f.id === selectedIds[0])
    : null;

  const openRename = () => {
    if (!selectedItem) return;
    setRenameName(selectedItem.data?.name || "");
    setShowRenameModal(true);
  };

  const onRenameSubmit = async () => {
    const name = renameName.trim();
    if (!selectedItem || !name) return;
    try {
      await apiservice.updateDocument({
        documentId: selectedItem.id,
        name,
        parentId: currentParentId,
      });
      toast.success(`Renamed to ${name}`);
      setShowRenameModal(false);
      setSelectedIds([]);
      onUpdate();
    } catch (e) {
      toast.error(`Failed to rename: ${e.message || e}`);
    }
  };

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
        {!isTrash && <Button size="sm" variant="outline" onClick={openRename} disabled={selectedIds.length !== 1}>Rename</Button>}
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

      <Modal show={showRenameModal} onHide={() => setShowRenameModal(false)}>
        <Modal.Header closeButton>
          Rename
        </Modal.Header>

        <Modal.Body>
          <Form onSubmit={(e) => { e.preventDefault(); onRenameSubmit(); }}>
            <InputGroup className="mb-3">
              <Form.Control
                autoFocus={true}
                type="text"
                value={renameName}
                onChange={(e) => setRenameName(e.currentTarget.value)}
              />
              <Button type="submit" variant="primary" disabled={!renameName.trim()}>Rename</Button>
            </InputGroup>
          </Form>
        </Modal.Body>
      </Modal>
    </>
  );
}
