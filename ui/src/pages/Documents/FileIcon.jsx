import {
  BsFilePdf,
  BsFolder2,
  BsFolder2Open,
  BsFileEarmark,
  BsBook,
  BsJournalText,
  BsCloud,
  BsTrash,
  BsHouseDoor,
} from "react-icons/bs";

// FileIcon renders the icon for a tree/list entry. `isOpen` only matters for
// folders, so their glyph reflects the tree's expanded/collapsed state.
export default function FileIcon({ file, isOpen }) {
  const Icon = () => {
    // Synthetic top-level nodes carry an explicit icon hint.
    switch (file.icon) {
      case "device":
        return <BsHouseDoor />;
      case "trash":
        return <BsTrash />;
      case "cloud":
        return <BsCloud />;
      default:
        break;
    }

    if (file.isFolder) {
      return isOpen ? <BsFolder2Open /> : <BsFolder2 />;
    }

    switch (file.type) {
      case "pdf":
        return <BsFilePdf />;
      case "epub":
        return <BsBook />;
      case "notebook":
        return <BsJournalText />;
      default:
        return <BsFileEarmark />;
    }
  };

  return (
    <span className="fileicon" style={{ padding: "0 0.4em 0 0" }}>
      <Icon />
    </span>
  );
}
