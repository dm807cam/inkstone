import { BsChevronRight } from "react-icons/bs";
import { Button } from "react-bootstrap";

const INTERNAL_ROOT = "__REACT_ARBORIST_INTERNAL_ROOT__";

export default function NameTag({ node, onSelect }) {
    if (!node || node.id === INTERNAL_ROOT || !node.data?.name) {
        return null;
    }

    const crumb = (
        <Button variant="outline" size="sm" onClick={() => onSelect(node)}>
            {node.data.name}
        </Button>
    );

    if (node.parent && node.parent.id !== INTERNAL_ROOT && node.parent.data?.name) {
        return (
            <>
                <NameTag node={node.parent} onSelect={onSelect} />
                <BsChevronRight />
                {crumb}
            </>
        );
    }

    return crumb;
}
