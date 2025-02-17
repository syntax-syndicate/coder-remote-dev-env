import AddIcon from "@mui/icons-material/AddOutlined";
import Inventory2 from "@mui/icons-material/Inventory2";
import NoteAddOutlined from "@mui/icons-material/NoteAddOutlined";
import UploadOutlined from "@mui/icons-material/UploadOutlined";
import { Button } from "components/Button/Button";
import {
	DropdownMenu,
	DropdownMenuContent,
	DropdownMenuItem,
	DropdownMenuTrigger,
} from "components/DropdownMenu/DropdownMenu";
import type { FC } from "react";

type CreateTemplateButtonProps = {
	onNavigate: (path: string) => void;
};

export const CreateTemplateButton: FC<CreateTemplateButtonProps> = ({
	onNavigate,
}) => {
	return (
		<DropdownMenu>
			<DropdownMenuTrigger asChild>
				<Button>
					<AddIcon />
					Create Template
				</Button>
			</DropdownMenuTrigger>
			<DropdownMenuContent align="end">
				<DropdownMenuItem
					onClick={() => onNavigate("/templates/new?exampleId=scratch")}
				>
					<NoteAddOutlined />
					From scratch
				</DropdownMenuItem>
				<DropdownMenuItem onClick={() => onNavigate("/templates/new")}>
					<UploadOutlined />
					Upload template
				</DropdownMenuItem>
				<DropdownMenuItem onClick={() => onNavigate("/starter-templates")}>
					<Inventory2 />
					Choose a starter template
				</DropdownMenuItem>
			</DropdownMenuContent>
		</DropdownMenu>
	);
};
